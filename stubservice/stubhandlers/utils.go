package stubhandlers

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/golang/groupcache/singleflight"
	"github.com/mozilla-services/stubattribution/stubmodify"
	"github.com/mozilla-services/stubattribution/stubservice/metrics"
	"github.com/pkg/errors"
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

	res := &stub{
		body:        data,
		contentType: resp.Header.Get("Content-Type"),
		filename:    path.Base(resp.Request.URL.Path),
	}
	globalStubCache.Add(url, res.copy())

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

func modifyStub(st *stub, attributionCode string) (res *stub, err error) {
	metrics.Statsd.Increment("modify_stub")

	body := st.body
	if attributionCode != "" {
		if body, err = stubmodify.WriteAttributionCode(st.body, []byte(attributionCode)); err != nil {
			return nil, errors.Wrapf(err, "WriteAttributionCode code: %s", attributionCode)
		}
	}
	return &stub{
		body:        body,
		contentType: st.contentType,
	}, nil
}
