package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	raven "github.com/getsentry/raven-go"
	"github.com/mozilla-services/go-stubattribution/stubservice/stubhandlers"
)

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

func main() {
	var stubHandler stubhandlers.StubHandler
	if returnMode == "redirect" {
		stubHandler = &stubhandlers.StubHandlerRedirect{
			CDNPrefix: cdnPrefix,
			S3Bucket:  s3Bucket,
			S3Prefix:  s3Prefix,
		}
	} else {
		stubHandler = &stubhandlers.StubHandlerDirect{}
	}

	stubService := &stubhandlers.StubService{
		Handler:     stubHandler,
		RavenClient: ravenClient,
	}

	mux := http.NewServeMux()
	mux.Handle("/", stubService)

	log.Fatal(http.ListenAndServe("127.0.0.1:8000", mux))
}
