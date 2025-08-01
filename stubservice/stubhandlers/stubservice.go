package stubhandlers

import (
	"net/http"

	"github.com/mozilla-services/gostatsd/statsd"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/mozilla-services/stubattribution/stubservice/metrics"
	"github.com/sirupsen/logrus"
)

type stubService struct {
	Handler StubHandler

	AttributionCodeValidator *attributioncode.Validator

	BouncerBaseURL string
}

func NewStubService(stubHandler StubHandler, validator *attributioncode.Validator, bouncerBaseURL string) http.Handler {
	return &stubService{
		Handler:                  stubHandler,
		AttributionCodeValidator: validator,
		BouncerBaseURL:           bouncerBaseURL,
	}
}

func (s *stubService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer metrics.Statsd.NewTiming().Send("request.time")
	defer metrics.Statsd.Increment("request.count")

	query := req.URL.Query()

	redirectBouncer := func() {
		backupURL := bouncerURL(query.Get("product"), query.Get("lang"), query.Get("os"), s.BouncerBaseURL)
		http.Redirect(w, req, backupURL, http.StatusFound)
	}

	attributionCode := query.Get("attribution_code")
	code, err := s.AttributionCodeValidator.Validate(attributionCode, query.Get("attribution_sig"), req.Header.Get("Referer"))
	if err != nil {
		defer metrics.Statsd.Clone(statsd.Tags("error_type", "validation")).Increment("request.error")
		redirectBouncer()
		return
	}

	logrus.WithFields(
		logrus.Fields{
			"log_type": "download_started",
			"dltoken":  code.DownloadToken(),
			// We need to keep the next two fields until we are sure that the
			// consumers of this log statement use the new field for GA4
			// (`client_id_ga4`).
			//
			// See: https://github.com/mozilla-services/stubattribution/issues/155
			// See: https://github.com/mozilla-services/stubattribution/issues/209
			"visit_id":      code.ClientID,
			"client_id":     code.ClientID,
			"client_id_ga4": code.ClientIDGA4,
			"session_id":    code.SessionID,
			// See: https://bugzilla.mozilla.org/show_bug.cgi?id=1972893
			"dlsource": code.DownloadSource,
			// See: https://bugzilla.mozilla.org/show_bug.cgi?id=1896957
			"product": query.Get("product"),
			"os":      query.Get("os"),
			"lang":    query.Get("lang"),
		},
	).Info("Download Started")

	err = s.Handler.ServeStub(w, req, code)
	if err != nil {
		logEntry := logrus.WithError(err).WithFields(
			logrus.Fields{
				"url":         req.URL.String(),
				"product":     query.Get("product"),
				"os":          query.Get("os"),
				"lang":        query.Get("lang"),
				"code_record": code,
			},
		)

		errorType := "stub"
		switch err := err.(type) {
		case *modifyStubError:
			errorType = "modifystub"
			logEntry = logEntry.WithField("code", err.Code)
		case *fetchStubError:
			errorType = "fetchstub"
			logEntry = logEntry.WithField("status_code", err.StatusCode).WithField("fetch_stub_url", err.URL)
		}

		defer metrics.Statsd.Clone(statsd.Tags("error_type", errorType)).Increment("request.error")
		logEntry.WithField("error_type", errorType).Error("Error serving stub")

		redirectBouncer()
		return
	}

	logrus.WithFields(
		logrus.Fields{
			"log_type": "download_finished",
			"dltoken":  code.DownloadToken(),
			// We need to keep the next two fields until we are sure that the
			// consumers of this log statement use the new field for GA4
			// (`client_id_ga4`).
			//
			// See: https://github.com/mozilla-services/stubattribution/issues/155
			// See: https://github.com/mozilla-services/stubattribution/issues/209
			"visit_id":      code.ClientID,
			"client_id":     code.ClientID,
			"client_id_ga4": code.ClientIDGA4,
			"session_id":    code.SessionID,
			// See: https://bugzilla.mozilla.org/show_bug.cgi?id=1972893
			"dlsource": code.DownloadSource,
			// See: https://bugzilla.mozilla.org/show_bug.cgi?id=1896957
			"product": query.Get("product"),
			"os":      query.Get("os"),
			"lang":    query.Get("lang"),
		},
	).Info("Download Finished")
}

func trimToLen(s string, l int) string {
	if l < 0 || len(s) <= l {
		return s
	}
	return s[:l]
}
