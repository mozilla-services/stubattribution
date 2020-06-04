package stubhandlers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/golang/groupcache/singleflight"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/mozilla-services/stubattribution/stubservice/metrics"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	bURL := bouncerURL(product, lang, os)

	cdnURL, err := redirectResponse(bURL)
	if err != nil {
		return errors.Wrap(err, "redirectResponse")
	}

	filename, err := url.QueryUnescape(path.Base(cdnURL))
	if err != nil {
		return errors.Wrap(err, "QueryUnescape")
	}

	key := (s.KeyPrefix + "builds/" +
		s3PathEscape(product) + "/" +
		s3PathEscape(lang) + "/" +
		s3PathEscape(os) + "/" +
		uniqueKey(cdnURL, attributionCode) + "/" +
		filename)

	if !s.Storage.Exists(key) {
		_, err := s.sfGroup.Do(key, func() (interface{}, error) {
			stub, err := fetchStub(bURL)
			if err != nil {
				return nil, err
			}

			stub, err = modifyStub(stub, attributionCode)
			if err != nil {
				return nil, err
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

	stubLocation := s.CDNPrefix + key
	stubLocationUrl, err := url.Parse(stubLocation)
	if err != nil {
		return errors.Wrap(err, "url.Parse")
	}
	// Cache response for one day
	w.Header().Set("Cache-Control", "max-age=86400")
	http.Redirect(w, req, stubLocationUrl.String(), http.StatusFound)
	logrus.WithFields(logrus.Fields{
		"req_url":  req.URL.String(),
		"location": stubLocation}).Info("Redirected request")

	return nil
}

// redirectResponse returns "", nil if not found
func redirectResponse(url string) (string, error) {
	cacheKey := "redirectResponse:" + url
	if cdnURL := globalStringCache.Get(cacheKey); cdnURL != "" {
		metrics.Statsd.Increment("redirect_response.cache_hit")
		return cdnURL, nil
	}

	defer metrics.Statsd.NewTiming().Send("redirect_response.time")
	metrics.Statsd.Increment("redirect_response.cache_miss")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrapf(err, "NewRequest url: %s", url)
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", errors.Wrapf(err, "RoundTrip")
	}
	defer resp.Body.Close()

	cdnURL := resp.Header.Get("Location")
	switch {
	case resp.StatusCode != 302:
		return "", errors.Errorf("url: %s returned %d, expecting 302", url, resp.StatusCode)
	case cdnURL == "":
		return "", errors.Errorf("url: %s returned 302, but Location was empty", url)
	}

	logrus.WithFields(logrus.Fields{
		"bouncer_url": url,
		"cdn_url":     cdnURL}).Info("Got redirect response")

	globalStringCache.Add(cacheKey, cdnURL)

	return cdnURL, nil
}

func s3PathEscape(key string) string {
	if key == "" {
		return "-"
	}
	return regexp.MustCompile("[^a-zA-Z0-9]").ReplaceAllString(key, "-")
}

func uniqueKey(downloadURL, attributionCode string) string {
	hasher := sha256.New()
	hasher.Write([]byte(downloadURL + "|" + attributionCode))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
