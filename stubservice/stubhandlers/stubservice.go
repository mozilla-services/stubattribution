package stubhandlers

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/alexcesaro/statsd"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/mozilla-services/stubattribution/stubservice/metrics"
)

type stubService struct {
	Handler StubHandler

	AttributionCodeValidator *attributioncode.Validator
}

func NewStubService(stubHandler StubHandler, validator *attributioncode.Validator) http.Handler {
	return &stubService{
		Handler:                  stubHandler,
		AttributionCodeValidator: validator,
	}
}

func (s *stubService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer metrics.Statsd.NewTiming().Send("request.time")
	defer metrics.Statsd.Increment("request.count")

	query := req.URL.Query()

	redirectBouncer := func() {
		backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"))
		http.Redirect(w, req, backupURL, http.StatusFound)
	}

	attributionCode := query.Get("attribution_code")
	code, err := s.AttributionCodeValidator.Validate(attributionCode, query.Get("attribution_sig"))
	if err != nil {
		defer metrics.Statsd.Clone(statsd.Tags("error_type", "validation")).Increment("request.error")
		logrus.WithError(err).WithField("attribution_code", trimToLen(attributionCode, 200)).Error("Could not validate attribution_code")
		redirectBouncer()
		return
	}

	err = s.Handler.ServeStub(w, req, code)
	if err != nil {
		defer metrics.Statsd.Clone(statsd.Tags("error_type", "stub")).Increment("request.error")
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
