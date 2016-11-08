package stubhandlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/mozilla-services/stubattribution/stubmodify"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/pkg/errors"

	raven "github.com/getsentry/raven-go"
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
		return "", errors.Wrap(err, "ParseQuery")
	}
	for k := range vals {
		if !validAttributionKeys[k] {
			return "", errors.Errorf("%s is not a valid attribution key", k)
		}
	}
	if len(vals) != len(validAttributionKeys) {
		return "", errors.New("code is missing keys")
	}
	return vals.Encode(), nil
}

func fetchModifyStub(url, attributionCode string) (*modifiedStub, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "http.Get url: %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.Errorf("url returned %d expecting 200", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll")
	}

	if attributionCode != "" {
		data, err = stubmodify.WriteAttributionCode(data, []byte(attributionCode))
		if err != nil {
			return nil, errors.Wrapf(err, "WriteAttributionCode code: %s", attributionCode)
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
		return "", errors.Wrapf(err, "NewRequest url: %s", url)
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", errors.Wrapf(err, "RoundTrip")
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode != 302:
		return "", errors.Errorf("url: %s returned %d, expecting 302", url, resp.StatusCode)
	case resp.Header.Get("Location") == "":
		return "", errors.Errorf("url: %s returned 302, but Location was empty", url)
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
		return errors.Wrap(err, "fetchModifyStub")
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
		return errors.Wrap(err, "redirectResponse")
	}

	filename, err := url.QueryUnescape(path.Base(cdnURL))
	if err != nil {
		return errors.Wrap(err, "QueryUnescape")
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
			return errors.Wrap(err, "fetchModifyStub")
		}

		err = s.Storage.Put(key, stub.Resp.Header.Get("Content-Type"), bytes.NewReader(stub.Data))
		if err != nil {
			return errors.Wrapf(err, "Put key: %s", key)
		}
	}

	// Cache response for one day
	w.Header().Set("Cache-Control", "max-age=86400")
	http.Redirect(w, req, s.CDNPrefix+key, http.StatusTemporaryRedirect)
	return nil
}

func checkMAC(msg, msgMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	expectedMac := mac.Sum(nil)
	return hmac.Equal(msgMAC, expectedMac)
}

// StubService serves redirects or modified stubs
type StubService struct {
	Handler StubHandler

	HMacKey string

	RavenClient *raven.Client
}

func (s *StubService) validateSignature(code, sig string) bool {
	// If no key is set, always succeed
	if s.HMacKey == "" {
		return true
	}

	byteSig, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	return checkMAC([]byte(code), byteSig, []byte(s.HMacKey))
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
			raven.CaptureMessage(fmt.Sprintf("%v", err), map[string]string{
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
		handleError(errors.Wrapf(err, "validateAttributionCode: code: %v", query.Get("attribution_code")))
		return
	}

	sig := query.Get("attribution_sig")
	if !s.validateSignature(code, sig) {
		handleError(errors.Errorf("signature not valid sig: %s, code: %s", sig, code))
		return
	}

	err = s.Handler.ServeStub(w, req, code)
	if err != nil {
		handleError(errors.Wrap(err, "ServeStub"))
		return
	}
}
