package stubhandlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	raven "github.com/getsentry/raven-go"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/pkg/errors"
)

// StubService serves redirects or modified stubs
type StubService struct {
	Handler StubHandler

	AttributionCodeValidator *attributioncode.Validator

	RavenClient *raven.Client
}

func (s *StubService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	query, _ := parseQueryNoEscape(req.URL.RawQuery)

	redirectBouncer := func() {
		backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"))
		http.Redirect(w, req, backupURL, http.StatusFound)
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

	code, err := s.AttributionCodeValidator.Validate(query.Get("attribution_code"), query.Get("attribution_sig"))
	if err != nil {
		handleError(errors.Wrapf(err, "could not validate attribution_code: %s", trimToLen(query.Get("attribution_code"), 200)))
		return
	}

	err = s.Handler.ServeStub(w, req, code.Encode())
	if err != nil {
		handleError(errors.Wrap(err, "ServeStub"))
		return
	}
}

// taken from net/url.go:parseQuery
func parseQueryNoEscape(query string) (m url.Values, err error) {
	m = make(url.Values)
	for query != "" {
		key := query
		if i := strings.IndexAny(key, "&;"); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}
		if key == "" {
			continue
		}
		value := ""
		if i := strings.Index(key, "="); i >= 0 {
			key, value = key[:i], key[i+1:]
		}
		key, err1 := url.QueryUnescape(key)
		if err1 != nil {
			if err == nil {
				err = err1
			}
			continue
		}

		if key != "attribution_code" {
			value, err1 = url.QueryUnescape(value)
			if err1 != nil {
				if err == nil {
					err = err1
				}
				continue
			}
		}

		m[key] = append(m[key], value)
	}
	return m, err
}

func trimToLen(s string, l int) string {
	if l < 0 || len(s) <= l {
		return s
	}
	return s[:l]
}
