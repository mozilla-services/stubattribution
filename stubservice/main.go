package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.mozilla.org/mozlogrus"

	"github.com/Sirupsen/logrus"
	"github.com/alexcesaro/statsd"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/evalphobia/logrus_sentry"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/mozilla-services/stubattribution/stubservice/metrics"
	"github.com/mozilla-services/stubattribution/stubservice/stubhandlers"
	"github.com/oremj/asyncstatsd"
)

const hmacTimeoutDefault = 10 * time.Minute

var (
	hmacKey        = os.Getenv("HMAC_KEY")
	hmacTimeoutEnv = os.Getenv("HMAC_TIMEOUT")
	hmacTimeout    = hmacTimeoutDefault

	returnMode = os.Getenv("RETURN_MODE")

	awsSess  = session.Must(session.NewSession())
	s3Bucket = os.Getenv("S3_BUCKET")
	s3Prefix = os.Getenv("S3_PREFIX")

	statsdPrefix = os.Getenv("STATSD_PREFIX")
	statsdAddr   = os.Getenv("STATSD_ADDR")

	cdnPrefix = os.Getenv("CDN_PREFIX")

	addr = os.Getenv("ADDR")

	sentryDSN = os.Getenv("SENTRY_DSN")
)

func mustStatsd(opts ...statsd.Option) *statsd.Client {
	c, err := statsd.New(opts...)
	if err != nil {
		logrus.WithError(err).Fatal("Could not initiate statsd")
	}
	return c
}

func init() {
	mozlogrus.Enable("StubAttribution")

	switch returnMode {
	case "redirect":
		returnMode = "redirect"
	default:
		returnMode = "direct"
	}

	if cdnPrefix == "" {
		cdnPrefix = fmt.Sprintf("https://s3.amazonaws.com/%s/", s3Bucket)
	}

	if addr == "" {
		addr = "127.0.0.1:8000"
	}
	if sentryDSN != "" {
		hook, err := logrus_sentry.NewSentryHook(sentryDSN, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})
		if err != nil {
			logrus.WithError(err).Fatal("Could not create raven client")
		}
		// Don't wait for sentry errors.
		hook.Timeout = 0
		logrus.AddHook(hook)
	}

	if hmacTimeoutEnv != "" {
		d, err := time.ParseDuration(hmacTimeoutEnv)
		if err != nil {
			logrus.WithError(err).Fatal("Could not parse HMAC_TIMEOUT")
		}
		hmacTimeout = d
	}

	if os.Getenv("AWS_REGION") == "" {
		meta := ec2metadata.New(awsSess)
		if region, _ := meta.Region(); region != "" {
			awsSess = awsSess.Copy(&aws.Config{Region: aws.String(region)})
		}
	}

	if statsdAddr == "" {
		statsdAddr = "127.0.0.1:8125"
	}
	if statsdPrefix == "" {
		statsdPrefix = "stubattribution"
	}
	metrics.Statsd = asyncstatsd.New(mustStatsd(
		statsd.Prefix(statsdPrefix),
		statsd.Address(statsdAddr),
		statsd.TagsFormat(statsd.Datadog),
	), 10000)
}

func okHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}

var versionFilePath = "/app/version.json"

func versionHandler(w http.ResponseWriter, req *http.Request) {
	versionFile, err := ioutil.ReadFile(versionFilePath)
	if err != nil {
		logrus.WithError(err).Errorf("Could not read %s", versionFilePath)
		http.Error(w, "Could not read version file.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(versionFile)
}

func main() {
	var stubHandler stubhandlers.StubHandler
	if returnMode == "redirect" {
		logrus.WithFields(logrus.Fields{
			"bucket": s3Bucket + s3Prefix,
			"cdn":    cdnPrefix,
		}).Info("Starting in redirect mode")
		storage := backends.NewS3(s3.New(awsSess), s3Bucket)
		stubHandler = stubhandlers.NewRedirectHandler(storage, cdnPrefix, s3Prefix)
	} else {
		logrus.Info("Starting in direct mode")
		stubHandler = stubhandlers.NewDirectHandler()
	}

	stubService := stubhandlers.NewStubService(
		stubHandler,
		attributioncode.NewValidator(hmacKey, hmacTimeout),
	)

	mux := http.NewServeMux()
	mux.Handle("/", stubService)
	mux.HandleFunc("/__lbheartbeat__", okHandler)
	mux.HandleFunc("/__heartbeat__", okHandler)
	mux.HandleFunc("/__version__", versionHandler)

	logrus.Fatal(http.ListenAndServe(addr, mux))
}
