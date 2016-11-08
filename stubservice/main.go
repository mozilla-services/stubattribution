package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	raven "github.com/getsentry/raven-go"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
	"github.com/mozilla-services/stubattribution/stubservice/stubhandlers"
)

var hmacKey = os.Getenv("HMAC_KEY")

var returnMode = os.Getenv("RETURN_MODE")

var s3Bucket = os.Getenv("S3_BUCKET")
var s3Prefix = os.Getenv("S3_PREFIX")

var cdnPrefix = os.Getenv("CDN_PREFIX")

var addr = os.Getenv("ADDR")

var sentryDSN = os.Getenv("SENTRY_DSN")
var ravenClient *raven.Client

func init() {
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
		RavenClient: ravenClient,
	}

	mux := http.NewServeMux()
	mux.Handle("/", stubService)
	mux.HandleFunc("/__lbheartbeat__", okHandler)
	mux.HandleFunc("/__heartbeat__", okHandler)
	mux.HandleFunc("/__version__", versionHandler)

	log.Fatal(http.ListenAndServe(addr, mux))
}
