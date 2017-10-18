package stubhandlers

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/golang/groupcache/singleflight"
	"github.com/mozilla-services/stubattribution/stubmodify"
	"github.com/mozilla-services/stubattribution/stubservice/metrics"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var stubClient = &http.Client{
	Timeout: 30 * time.Second,
}

// BouncerURL is the base bouncer URL
var BouncerURL = "https://download.mozilla.org/"

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

type fetchStubError struct {
	error
	URL        string
	StatusCode int
}

// uses global stub cache
func fetchStub(url string) (*stub, error) {
	if s := globalStubCache.Get(url); s != nil {
		metrics.Statsd.Increment("fetch_stub.cache_hit")
		return s, nil
	}

	defer metrics.Statsd.NewTiming().Send("fetch_stub.time")
	metrics.Statsd.Increment("fetch_stub.cache_miss")

	resp, err := stubClient.Get(url)
	if err != nil {
		return nil, &fetchStubError{errors.Wrap(err, "Get"), url, 0}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, &fetchStubError{errors.New("invalid status code"), url, resp.StatusCode}
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, &fetchStubError{errors.Wrap(err, "ReadAll"), url, resp.StatusCode}
	}

	res := &stub{
		body:        data,
		contentType: resp.Header.Get("Content-Type"),
		filename:    path.Base(resp.Request.URL.Path),
	}
	globalStubCache.Add(url, res.copy())

	logrus.WithFields(logrus.Fields{
		"bouncer_url": url,
		"stub_size":   len(res.body),
		"stub_url":    resp.Request.URL.Path}).Info("Fetched stub")

	return res, nil
}

// sfFetchStub runs fetchStub in a singleflight group
func sfFetchStub(sfGroup *singleflight.Group, url string) (*stub, error) {
	res, err := sfGroup.Do(url, func() (interface{}, error) {
		return fetchStub(url)
	})
	if res == nil {
		return nil, err
	}
	return res.(*stub), err
}

type modifyStubError struct {
	error
	Code string
}

func modifyStub(st *stub, attributionCode string) (res *stub, err error) {
	metrics.Statsd.Increment("modify_stub")

	body := st.body
	if attributionCode != "" {
		if body, err = stubmodify.WriteAttributionCode(st.body, []byte(attributionCode)); err != nil {
			return nil, &modifyStubError{err, attributionCode}
		}
	}
	logrus.WithFields(logrus.Fields{
		"original_filename":    st.filename,
		"original_stub_sha256": fmt.Sprintf("%X", sha256.Sum256(st.body)),
		"modified_stub_sha256": fmt.Sprintf("%X", sha256.Sum256(body)),
		"attribution_code":     attributionCode}).Info("Modified stub")

	return &stub{
		body:        body,
		contentType: st.contentType,
	}, nil
}
