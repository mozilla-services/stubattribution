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

var BOUNCER_URL = "https://download.mozilla.org/"

func uniqueKey(downloadURL, attributionCode string) string {
	hasher := sha256.New()
	hasher.Write([]byte(downloadURL + "|" + attributionCode))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func bouncerURL(product, lang, os string) string {
	v := url.Values{}
	v.Set("product", product)
	v.Set("lang", lang)
	v.Set("os", os)
	return BOUNCER_URL + "?" + v.Encode()
}

type modifiedStub struct {
	Data []byte
	Resp *http.Response
}

func fetchModifyStub(url, attributionCode string) (*modifiedStub, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetchModifyStub: http.Get%v", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchModifyStub: %v", err)
	}

	if attributionCode != "" {
		data, err = stubattribution.WriteAttributionCode(data, []byte(attributionCode))
		if err != nil {
			return nil, fmt.Errorf("fetchModifyStub: %v", err)
		}
	}
	return &modifiedStub{
		Data: data,
		Resp: resp,
	}, nil

}

type StubHandler struct {
	ReturnMode string

	CDNPrefix string

	S3Bucket string
	S3Prefix string
}

// redirectResponse returns "", nil if not found
func redirectResponse(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("StubHandler: NewRequest: %v", err)
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", fmt.Errorf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 302 || resp.Header.Get("Location") == "" {
		return "", nil
	}

	return resp.Header.Get("Location"), nil
}

func (s *StubHandler) ServeDirect(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := query.Get("attribution_code")

	cdnURL, err := redirectResponse(bouncerURL(product, lang, os))
	if err != nil {
		log.Printf("StubHandler: redirectResponse: %v", err)
		http.Error(w, "Internal Service Error", http.StatusInternalServerError)
		return
	}
	if cdnURL == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	stub, err := fetchModifyStub(cdnURL, attributionCode)
	if err != nil {
		log.Printf("StubHandler: %v", err)
		http.Error(w, "Internal Service Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", stub.Resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(stub.Data)))
	w.Write(stub.Data)
}

func (s *StubHandler) ServeRedirect(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := query.Get("attribution_code")

	cdnURL, err := redirectResponse(bouncerURL(product, lang, os))
	if err != nil {
		log.Printf("StubHandler: redirectResponse: %v", err)
		http.Error(w, "Internal Service Error", http.StatusInternalServerError)
		return
	}
	if cdnURL == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	filename, err := url.QueryUnescape(path.Base(cdnURL))
	if err != nil {
		log.Printf("StubHandler: %v", err)
		http.Error(w, "Internal Service Error", http.StatusInternalServerError)
		return
	}

	s3Key := (s.S3Prefix + "builds/" +
		product + "/" +
		lang + "/" +
		os + "/" +
		uniqueKey(cdnURL, attributionCode) + "/" +
		filename)

	s3Svc := s3.New(session.New())
	_, err = s3Svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.S3Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		stub, err := fetchModifyStub(cdnURL, attributionCode)
		if err != nil {
			log.Printf("StubHandler: %v", err)
			http.Error(w, "Internal Service Error", http.StatusInternalServerError)
			return
		}
		putObjectParams := &s3.PutObjectInput{
			Bucket:      aws.String(s.S3Bucket),
			Key:         aws.String(s3Key),
			ContentType: aws.String(stub.Resp.Header.Get("Content-Type")),
			Body:        bytes.NewReader(stub.Data),
		}
		_, err = s3Svc.PutObject(putObjectParams)
		if err != nil {
			log.Printf("StubHandler: PutObject %v", err)
			http.Error(w, "Internal Service Error", http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, req, s.CDNPrefix+s3Key, http.StatusTemporaryRedirect)
}

func (s *StubHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.ReturnMode == "redirect" {
		s.ServeRedirect(w, req)
		return
	}
	s.ServeDirect(w, req)
}
