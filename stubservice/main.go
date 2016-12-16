package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mozilla.org/mozlog"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	raven "github.com/getsentry/raven-go"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/mozilla-services/stubattribution/stubservice/stubhandlers"
)

const hmacTimeoutDefault = 10 * time.Minute

var (
	hmacKey        = os.Getenv("HMAC_KEY")
	hmacTimeoutEnv = os.Getenv("HMAC_TIMEOUT_SECONDS")
	hmacTimeout    time.Duration

	returnMode = os.Getenv("RETURN_MODE")

	s3Bucket = os.Getenv("S3_BUCKET")
	s3Prefix = os.Getenv("S3_PREFIX")

	cdnPrefix = os.Getenv("CDN_PREFIX")

	addr = os.Getenv("ADDR")

	sentryDSN   = os.Getenv("SENTRY_DSN")
	ravenClient *raven.Client
)

func init() {
	mozlog.Logger.LoggerName = "StubAttribution"

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
		var err error
		ravenClient, err = raven.New(sentryDSN)
		if err != nil {
			log.Printf("SetDSN: %v", err)
			ravenClient = nil
		}
	}

	d, err := strconv.Atoi(hmacTimeoutEnv)
	if err != nil {
		hmacTimeout = hmacTimeoutDefault
	} else {
		hmacTimeout = time.Duration(d) * time.Second
	}
}

func okHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}

var versionFilePath = "/app/version.json"

func versionHandler(w http.ResponseWriter, req *http.Request) {
	versionFile, err := ioutil.ReadFile(versionFilePath)
	if err != nil {
		log.Printf("Error reading %s err: %v", versionFilePath, err)
		http.Error(w, "Could not read version file.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(versionFile)
}

func main() {
	var stubHandler stubhandlers.StubHandler
	if returnMode == "redirect" {
		storage := backends.NewS3(s3.New(session.New()), s3Bucket)
		stubHandler = &stubhandlers.StubHandlerRedirect{
			CDNPrefix: cdnPrefix,
			Storage:   storage,
			KeyPrefix: s3Prefix,
		}
	} else {
		stubHandler = &stubhandlers.StubHandlerDirect{}
	}

	stubService := &stubhandlers.StubService{
		Handler:     stubHandler,
		HMacKey:     hmacKey,
		HMacTimeout: hmacTimeout,
		RavenClient: ravenClient,
	}

	mux := http.NewServeMux()
	mux.Handle("/", stubService)
	mux.HandleFunc("/__lbheartbeat__", okHandler)
	mux.HandleFunc("/__heartbeat__", okHandler)
	mux.HandleFunc("/__version__", versionHandler)

	log.Fatal(http.ListenAndServe(addr, mux))
}
