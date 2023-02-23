package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/storage"

	"go.mozilla.org/mozlogrus"

	"github.com/getsentry/sentry-go"
	sentrylogrus "github.com/getsentry/sentry-go/logrus"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/mozilla-services/stubattribution/stubservice/stubhandlers"
	"github.com/sirupsen/logrus"
)

const (
	hmacTimeoutDefault = 10 * time.Minute
	// versionFilePath is the path to the `version.json` file in the Docker container.
	versionFilePath = "/app/version.json"
)

var (
	baseURL = os.Getenv("BASE_URL")

	hmacKey        = os.Getenv("HMAC_KEY")
	hmacTimeoutEnv = os.Getenv("HMAC_TIMEOUT")
	hmacTimeout    = hmacTimeoutDefault

	returnMode = os.Getenv("RETURN_MODE")

	storageBackend = os.Getenv("STORAGE_BACKEND")

	gcsBucket = os.Getenv("GCS_BUCKET")
	gcsPrefix = os.Getenv("GCS_PREFIX")

	cdnPrefix = os.Getenv("CDN_PREFIX")

	addr = os.Getenv("ADDR")

	sentryDSN = os.Getenv("SENTRY_DSN")

	debugMode = false
)

func init() {
	mozlogrus.Enable("StubAttribution")

	if debug, err := strconv.ParseBool(os.Getenv("DEBUG_MODE")); err == nil && debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Debug mode is enabled")

		debugMode = true
	}

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
	case "gcs":
	default:
		logrus.Fatal("Invalid STORAGE_BACKEND value")
	}

	if cdnPrefix == "" {
		switch storageBackend {
		case "gcs":
			cdnPrefix = fmt.Sprintf("https://storage.googleapis.com/%s/", gcsBucket)
		}
	}

	if addr == "" {
		addr = "127.0.0.1:8000"
	}
	if sentryDSN != "" {
		// Send only ERROR and higher level logs to Sentry.
		sentryLevels := []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
		sentryOptions := sentry.ClientOptions{
			Dsn:              sentryDSN,
			Debug:            debugMode,
			AttachStacktrace: true,
		}

		// We attempt to read the `version.json` file in order to set the Sentry
		// "release". If something goes wrong, we do nothing, which will let Sentry
		// fallback to a default release value (retrieved with `git`).
		if versionJSON, err := ioutil.ReadFile(versionFilePath); err == nil {
			var version struct {
				Version string
			}
			if err := json.Unmarshal(versionJSON, &version); err == nil {
				logrus.Debugf("Using release from version.json file: %s", version.Version)
				sentryOptions.Release = version.Version
			}
		}

		hook, err := sentrylogrus.New(sentryLevels, sentryOptions)
		if err != nil {
			logrus.WithError(err).Fatal("Could not create sentry client")
		}

		defer hook.Flush(5 * time.Second)
		logrus.AddHook(hook)

		// Flushes before calling os.Exit(1) when using logger.Fatal (else all
		// defers are not called, and Sentry does not have time to send the event).
		logrus.RegisterExitHandler(func() { hook.Flush(5 * time.Second) })
	}

	if hmacTimeoutEnv != "" {
		d, err := time.ParseDuration(hmacTimeoutEnv)
		if err != nil {
			logrus.WithError(err).Fatal("Could not parse HMAC_TIMEOUT")
		}
		hmacTimeout = d
	}
}

func okHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}

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
	attrQuery.Set("experiment", "pingdom")
	attrQuery.Set("variation", "pingdom")
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
		if storageBackend == "gcs" {
			logrus.WithFields(logrus.Fields{
				"backend": storageBackend,
				"bucket":  gcsBucket,
				"prefix":  gcsPrefix,
				"cdn":     cdnPrefix,
			}).Info("Starting in redirect mode")

			gcsStorageClient, err := storage.NewClient(context.Background())
			if err != nil {
				logrus.WithError(err).Fatal("Could not create GCS storage client")
			}

			store := backends.NewGCS(gcsStorageClient, gcsBucket, time.Hour*24)
			stubHandler = stubhandlers.NewRedirectHandler(store, cdnPrefix, gcsPrefix)
		} else {
			logrus.WithField("backend", storageBackend).Fatal("Unsupported storage backend")
		}
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
