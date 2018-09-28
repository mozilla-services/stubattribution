package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/storage"

	"go.mozilla.org/mozlogrus"

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
	"github.com/sirupsen/logrus"
)

const hmacTimeoutDefault = 10 * time.Minute

var (
	baseURL = os.Getenv("BASE_URL")

	hmacKey        = os.Getenv("HMAC_KEY")
	hmacTimeoutEnv = os.Getenv("HMAC_TIMEOUT")
	hmacTimeout    = hmacTimeoutDefault

	returnMode = os.Getenv("RETURN_MODE")

	storageBackend = os.Getenv("STORAGE_BACKEND")

	s3Bucket = os.Getenv("S3_BUCKET")
	s3Prefix = os.Getenv("S3_PREFIX")

	gcsBucket = os.Getenv("GCS_BUCKET")
	gcsPrefix = os.Getenv("GCS_PREFIX")

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

func awsSess() *session.Session {
	awsSess := session.Must(session.NewSession())
	if os.Getenv("AWS_REGION") == "" {
		meta := ec2metadata.New(awsSess)
		if region, _ := meta.Region(); region != "" {
			awsSess = awsSess.Copy(&aws.Config{Region: aws.String(region)})
		}
	}
	return awsSess
}

func init() {
	mozlogrus.Enable("StubAttribution")

	if baseURL == "" {
		logrus.Fatal("BASE_URL is required")
	}

	switch returnMode {
	case "redirect":
		returnMode = "redirect"
	default:
		returnMode = "direct"
	}

	// Validate STORAGE_BACKEND
	switch storageBackend {
	case "", "s3":
		storageBackend = "s3"
	case "gcs":
	default:
		logrus.Fatal("Invalid STORAGE_BACKEND")

	}

	if cdnPrefix == "" {
		switch storageBackend {
		case "s3":
			cdnPrefix = fmt.Sprintf("https://s3.amazonaws.com/%s/", s3Bucket)
		case "gcs":
			cdnPrefix = fmt.Sprintf("https://storage.googleapis.com/%s/", gcsBucket)
		}

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

func pingdomHandler(w http.ResponseWriter, req *http.Request) {
	attrQuery := url.Values{}
	attrQuery.Set("source", "mozilla.com")
	attrQuery.Set("medium", "pingdom")
	attrQuery.Set("campaign", "pingdom")
	attrQuery.Set("content", "pingdom")
	b64AttrQuery := base64.URLEncoding.WithPadding('.').EncodeToString([]byte(attrQuery.Encode()))

	query := url.Values{}
	query.Set("product", "test-stub")
	query.Set("os", "win")
	query.Set("lang", "en-US")
	query.Set("attribution_code", b64AttrQuery)
	if hmacKey != "" {
		hasher := hmac.New(sha256.New, []byte(hmacKey))
		hasher.Write([]byte(b64AttrQuery))
		query.Set("attribution_sig", fmt.Sprintf("%x", hasher.Sum(nil)))
	}
	http.Redirect(w, req, baseURL+"?"+query.Encode(), http.StatusFound)
}

func main() {
	var stubHandler stubhandlers.StubHandler
	if returnMode == "redirect" {
		var store backends.Storage
		var storagePrefix string
		if storageBackend == "s3" {
			logrus.WithFields(logrus.Fields{
				"bucket": s3Bucket + s3Prefix,
				"cdn":    cdnPrefix,
			}).Info("Starting in redirect mode (backend: s3)")
			store = backends.NewS3(s3.New(awsSess()), s3Bucket, time.Hour*24)
			storagePrefix = s3Prefix
		} else if storageBackend == "gcs" {
			logrus.WithFields(logrus.Fields{
				"bucket": s3Bucket + s3Prefix,
				"cdn":    cdnPrefix,
			}).Info("Starting in redirect mode (backend: gcs)")
			gcsStorageClient, err := storage.NewClient(context.Background())
			if err != nil {
				logrus.WithError(err).Fatal("Could not create GCS storage client")
			}
			store = backends.NewGCS(gcsStorageClient, gcsBucket, time.Hour*24)
			storagePrefix = gcsPrefix
		} else {
			logrus.Fatal("Invalid STORAGE_BACKEND")
		}
		stubHandler = stubhandlers.NewRedirectHandler(store, cdnPrefix, storagePrefix)
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
	mux.HandleFunc("/__pingdom__", pingdomHandler)

	logrus.Fatal(http.ListenAndServe(addr, mux))
}
