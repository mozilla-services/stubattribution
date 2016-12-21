package stubhandlers

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/mozilla-services/stubattribution/stubmodify"
	"github.com/pkg/errors"
)

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
