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
	"github.com/mozilla-services/go-stubattribution/stubmodify"
)

// BouncerURL is the base bouncer URL
var BouncerURL = "https://download.mozilla.org/"

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
	return BouncerURL + "?" + v.Encode()
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
		data, err = stubmodify.WriteAttributionCode(data, []byte(attributionCode))
		if err != nil {
			return nil, fmt.Errorf("fetchModifyStub: %v", err)
		}
	}
	return &modifiedStub{
		Data: data,
		Resp: resp,
	}, nil

}

// StubHandler serves redirects or modified stubs
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

// ServeDirect serves stub bytes directly through handler
func (s *StubHandler) ServeDirect(w http.ResponseWriter, req *http.Request) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := query.Get("attribution_code")

	stub, err := fetchModifyStub(bouncerURL(product, lang, os), attributionCode)
	if err != nil {
		return fmt.Errorf("StubHandler: %v", err)
	}
	if stub.Resp.StatusCode != 200 {
		return fmt.Errorf("fetchModifyStub returned: %d", stub.Resp.StatusCode)
	}
	w.Header().Set("Content-Type", stub.Resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(stub.Data)))
	w.Write(stub.Data)
	return nil
}

// ServeRedirect redirects to modified stub
func (s *StubHandler) ServeRedirect(w http.ResponseWriter, req *http.Request) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := query.Get("attribution_code")

	cdnURL, err := redirectResponse(bouncerURL(product, lang, os))
	if err != nil {
		return fmt.Errorf("redirectResponse: %v", err)
	}

	if cdnURL == "" {
		return fmt.Errorf("redirectResponse: cdnURL was blank")
	}

	filename, err := url.QueryUnescape(path.Base(cdnURL))
	if err != nil {
		return fmt.Errorf("StubHandler: %v", err)
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
			return fmt.Errorf("fetchModifyStub: %v", err)
		}
		if stub.Resp.StatusCode != 200 {
			return fmt.Errorf("fetchModifyStub returned: %d", stub.Resp.StatusCode)
		}
		putObjectParams := &s3.PutObjectInput{
			Bucket:      aws.String(s.S3Bucket),
			Key:         aws.String(s3Key),
			ContentType: aws.String(stub.Resp.Header.Get("Content-Type")),
			Body:        bytes.NewReader(stub.Data),
		}
		_, err = s3Svc.PutObject(putObjectParams)
		if err != nil {
			return fmt.Errorf("StubHandler: PutObject %v", err)
		}
	}
	http.Redirect(w, req, s.CDNPrefix+s3Key, http.StatusTemporaryRedirect)
	return nil
}

func (s *StubHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"))

	if s.ReturnMode == "redirect" {
		err := s.ServeRedirect(w, req)
		if err != nil {
			log.Printf("ServeRedirect: %v", err)
			http.Redirect(w, req, backupURL, http.StatusTemporaryRedirect)
		}
		return
	}
	err := s.ServeDirect(w, req)
	if err != nil {
		log.Printf("ServeDirect: %v", err)
		http.Redirect(w, req, backupURL, http.StatusTemporaryRedirect)
	}
}
