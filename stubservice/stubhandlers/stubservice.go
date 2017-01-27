package stubhandlers

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/mozilla-services/stubattribution/attributioncode"
)

// StubService serves redirects or modified stubs
type StubService struct {
	Handler StubHandler

	AttributionCodeValidator *attributioncode.Validator
}

func (s *StubService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	redirectBouncer := func() {
		backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"))
		http.Redirect(w, req, backupURL, http.StatusFound)
	}

	attributionCode := query.Get("attribution_code")
	code, err := s.AttributionCodeValidator.Validate(attributionCode, query.Get("attribution_sig"))
	if err != nil {
		logrus.WithError(err).WithField("attribution_code", trimToLen(attributionCode, 200)).Error("Could not validate attribution_code")
		redirectBouncer()
		return
	}

	err = s.Handler.ServeStub(w, req, code)
	if err != nil {
		logrus.WithError(err).WithField("url", req.URL.String()).Error("Error serving stub")
		redirectBouncer()
		return
	}
}

func trimToLen(s string, l int) string {
	if l < 0 || len(s) <= l {
		return s
	}
	return s[:l]
}
