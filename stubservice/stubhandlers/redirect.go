package stubhandlers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/golang/groupcache/singleflight"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/pkg/errors"
)

// redirectHandler serves redirects to modified stub binaries
type redirectHandler struct {
	CDNPrefix string

	Storage backends.Storage

	KeyPrefix string

	sfGroup *singleflight.Group
}

// NewRedirectHandler returns a new StubHandler
func NewRedirectHandler(storage backends.Storage, cdnPrefix, keyPrefix string) StubHandler {
	return &redirectHandler{
		CDNPrefix: cdnPrefix,
		KeyPrefix: keyPrefix,

		Storage: storage,

		sfGroup: new(singleflight.Group),
	}
}

// ServeStub redirects to modified stub
func (s *redirectHandler) ServeStub(w http.ResponseWriter, req *http.Request, code string) error {
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
		_, err := s.sfGroup.Do(key, func() (interface{}, error) {
			stub, err := fetchStub(bouncerURL(product, lang, os))
			if err != nil {
				return nil, errors.Wrap(err, "fetchStub")
			}
			stub, err = modifyStub(stub, attributionCode)
			if err != nil {
				return nil, errors.Wrap(err, "modifyStub")
			}

			if err := s.Storage.Put(key, stub.contentType, bytes.NewReader(stub.body)); err != nil {
				return nil, errors.Wrapf(err, "Put key: %s", key)
			}
			return nil, nil
		})

		if err != nil {
			return err
		}
	}

	// Cache response for one day
	w.Header().Set("Cache-Control", "max-age=86400")
	http.Redirect(w, req, s.CDNPrefix+key, http.StatusFound)
	return nil
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

func uniqueKey(downloadURL, attributionCode string) string {
	hasher := sha256.New()
	hasher.Write([]byte(downloadURL + "|" + attributionCode))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
