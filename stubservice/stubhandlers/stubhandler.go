package stubhandlers

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	raven "github.com/getsentry/raven-go"
	"github.com/mozilla-services/go-stubattribution/stubmodify"
	"github.com/mozilla-services/go-stubattribution/stubservice/backends"
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

var validAttributionKeys = map[string]bool{
	"source":   true,
	"medium":   true,
	"campaign": true,
	"content":  true,
}

func validateAttributionCode(code string) (string, error) {
	if len(code) > 200 {
		return "", errors.New("code longer than 200 characters")
	}
	unEscapedCode, err := url.QueryUnescape(code)
	vals, err := url.ParseQuery(unEscapedCode)
	if err != nil {
		return "", fmt.Errorf("ParseQuery: %v", err)
	}
	for k := range vals {
		if !validAttributionKeys[k] {
			return "", fmt.Errorf("%s is not a valid attribution key", k)
		}
	}
	if len(vals) != len(validAttributionKeys) {
		return "", fmt.Errorf("code is missing keys")
	}
	return vals.Encode(), nil
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

// StubHandler interface returns an error if anything went wrong
// else it handled the request successfully
type StubHandler interface {
	ServeStub(http.ResponseWriter, *http.Request, string) error
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

// StubHandlerDirect serves modified stub binaries directly
type StubHandlerDirect struct {
}

// ServeStub serves stub bytes directly through handler
func (s *StubHandlerDirect) ServeStub(w http.ResponseWriter, req *http.Request, code string) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := code

	stub, err := fetchModifyStub(bouncerURL(product, lang, os), attributionCode)
	if err != nil {
		return fmt.Errorf("fetchModifyStub: %v", err)
	}
	if stub.Resp.StatusCode != 200 {
		return fmt.Errorf("fetchModifyStub returned: %d", stub.Resp.StatusCode)
	}

	// Cache response for one week
	w.Header().Set("Cache-Control", "max-age=604800")
	w.Header().Set("Content-Type", stub.Resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(stub.Data)))
	w.Write(stub.Data)
	return nil
}

// StubHandlerRedirect serves redirects to modified stub binaries
type StubHandlerRedirect struct {
	CDNPrefix string

	Storage *backends.S3

	KeyPrefix string
}

// ServeStub redirects to modified stub
func (s *StubHandlerRedirect) ServeStub(w http.ResponseWriter, req *http.Request, code string) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := code

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

	key := (s.KeyPrefix + "builds/" +
		product + "/" +
		lang + "/" +
		os + "/" +
		uniqueKey(cdnURL, attributionCode) + "/" +
		filename)

	if !s.Storage.Exists(key) {
		stub, err := fetchModifyStub(cdnURL, attributionCode)
		if err != nil {
			return fmt.Errorf("fetchModifyStub: %v", err)
		}
		if stub.Resp.StatusCode != 200 {
			return fmt.Errorf("fetchModifyStub returned: %d", stub.Resp.StatusCode)
		}
		err = s.Storage.Put(key, stub.Resp.Header.Get("Content-Type"), bytes.NewReader(stub.Data))
		if err != nil {
			return fmt.Errorf("Put %v", err)
		}
	}

	// Cache response for one day
	w.Header().Set("Cache-Control", "max-age=86400")
	http.Redirect(w, req, s.CDNPrefix+key, http.StatusTemporaryRedirect)
	return nil
}

// StubService serves redirects or modified stubs
type StubService struct {
	Handler StubHandler

	RavenClient *raven.Client
}

func (s *StubService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	redirectBouncer := func() {
		backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"))
		http.Redirect(w, req, backupURL, http.StatusTemporaryRedirect)
	}

	handleError := func(err error) {
		log.Println(err)
		if s.RavenClient != nil {
			raven.CaptureError(err, map[string]string{
				"url": req.URL.String(),
			})
		}
		redirectBouncer()
	}

	code := query.Get("attribution_code")
	if code == "" {
		redirectBouncer()
		return
	}

	code, err := validateAttributionCode(query.Get("attribution_code"))
	if err != nil {
		handleError(fmt.Errorf("validateAttributionCode: %v", err))
		return
	}

	err = s.Handler.ServeStub(w, req, code)
	if err != nil {
		handleError(fmt.Errorf("ServeStub: %v", err))
		return
	}
}
