package stubhandlers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mozilla-services/go-stubattribution"
)

var RETURN_METHOD = "redirect"
var BOUNCER_URL = "https://download.mozilla.org/"
var S3_BUCKET = "net-mozaws-stage-us-east-1-stub-attribution"
var CDN_PREFIX = fmt.Sprintf("https://s3.amazonaws.com/%s/", S3_BUCKET)

func uniqueKey(downloadURL, attributionCode string) string {
	hasher := sha256.New()
	hasher.Write([]byte(downloadURL + "|" + attributionCode))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func bouncerURL(product, lang, os string) string {
	v := url.Values{}
	v.Set("product", product)
	if lang != "" {
		v.Set("lang", lang)
	}
	if os != "" {
		v.Set("os", os)
	}
	return BOUNCER_URL + "?" + v.Encode()
}

func fetchModifyStub(url, attributionCode string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetchModifyStub: http.Get%v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchModifyStub: %v", err)
	}

	if attributionCode == "" {
		return body, nil
	}

	data, err := stubattribution.WriteAttributionCode(body, []byte(attributionCode))
	if err != nil {
		return nil, fmt.Errorf("fetchModifyStub: %v", err)
	}
	return data, nil
}

func StubHandler(w http.ResponseWriter, r *http.Request) {
	product := "Firefox-46.0-Stub"
	lang := "en-US"
	os := "win"
	attributionCode := ""

	req, err := http.NewRequest("GET",
		bouncerURL(product, lang, os), nil)
	if err != nil {
		log.Printf("StubHandler: NewRequest: %v", err)
		http.Error(w, "Internal Service Error", http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Printf("StubHandler: RoundTrip: %v", err)
		http.Error(w, "Internal Service Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 302 || resp.Header.Get("Location") == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	cdnURL := resp.Header.Get("Location")
	if RETURN_METHOD == "direct" {
		data, err := fetchModifyStub(cdnURL, attributionCode)
		if err != nil {
			log.Printf("StubHandler: %v", err)
			http.Error(w, "Internal Service Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
		return
	} else if RETURN_METHOD == "redirect" {
		filename, err := url.QueryUnescape(path.Base(cdnURL))
		if err != nil {
			log.Printf("StubHandler: %v", err)
			http.Error(w, "Internal Service Error", http.StatusInternalServerError)
			return
		}

		s3Key := ("builds/" +
			product + "/" +
			lang + "/" +
			os + "/" +
			uniqueKey(cdnURL, attributionCode) + "/" +
			filename)

		s3Svc := s3.New(session.New())
		_, err = s3Svc.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(S3_BUCKET),
			Key:    aws.String(s3Key),
		})
		if err != nil {
			data, err := fetchModifyStub(cdnURL, attributionCode)
			if err != nil {
				log.Printf("StubHandler: %v", err)
				http.Error(w, "Internal Service Error", http.StatusInternalServerError)
				return
			}
			putObjectParams := &s3.PutObjectInput{
				Bucket:      aws.String(S3_BUCKET),
				Key:         aws.String(s3Key),
				ContentType: aws.String(resp.Header.Get("Content-Type")),
				Body:        bytes.NewReader(data),
			}
			_, err = s3Svc.PutObject(putObjectParams)
			if err != nil {
				log.Printf("StubHandler: PutObject %v", err)
				http.Error(w, "Internal Service Error", http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, req, CDN_PREFIX+s3Key, http.StatusTemporaryRedirect)
	}
}
