package stubhandlers

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var validAttributionKeys = map[string]bool{
	"source":   true,
	"medium":   true,
	"campaign": true,
	"content":  true,
}

type attributionCodeQuery struct {
	Raw       string
	TimeStamp time.Time
	UrlVals   url.Values
}

func newAttributionCodeQuery(code string) (*attributionCodeQuery, error) {
	if len(code) > 200 {
		return nil, errors.New("code longer than 200 characters")
	}

	unEscapedCode, err := url.QueryUnescape(code)
	if err != nil {
		return nil, errors.Wrap(err, "QueryUnescape")
	}
	vals, err := url.ParseQuery(unEscapedCode)
	if err != nil {
		return nil, errors.Wrap(err, "ParseQuery")
	}

	timeStamp := time.Now()
	if ts := vals.Get("timestamp"); ts != "" {
		timeStamp, err = parseTimeStamp(ts)
		if err != nil {
			return nil, errors.Wrap(err, "parseTimeStamp")
		}
		vals.Del("timestamp")
	}

	query := &attributionCodeQuery{
		Raw:       code,
		TimeStamp: timeStamp,
		UrlVals:   vals,
	}

	if err := query.validate(); err != nil {
		return nil, errors.Wrap(err, "validate")
	}
	return query, nil
}

func (a *attributionCodeQuery) validate() error {
	for k := range a.UrlVals {
		if !validAttributionKeys[k] {
			return errors.Errorf("%s is not a valid attribution key", k)
		}
	}

	if len(a.UrlVals) != len(validAttributionKeys) {
		return errors.New("code is missing keys")
	}

	if source := a.UrlVals.Get("source"); !sourceWhitelist[source] {
		return fmt.Errorf("source: %s is not in whitelist", source)
	}

	return nil
}

func (a *attributionCodeQuery) validateSignature(key string, timeout time.Duration, sig string) bool {
	// If no key is set, always succeed
	if key == "" {
		return true
	}

	if time.Since(a.TimeStamp) > timeout {
		return false
	}

	byteSig, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	return checkMAC([]byte(a.Raw), byteSig, []byte(key))
}

func parseTimeStamp(ts string) (time.Time, error) {
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Now(), errors.Wrap(err, "Atoi")
	}
	return time.Unix(tsInt, 0), nil
}
