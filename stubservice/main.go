package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mozilla-services/go-stubattribution/stubservice/stubhandlers"
)

var RETURN_MODE = os.Getenv("RETURN_MODE")

var S3_BUCKET = os.Getenv("S3_BUCKET")
var S3_PREFIX = os.Getenv("S3_PREFIX")

var CDN_PREFIX = os.Getenv("CDN_PREFIX")

var ADDR = os.Getenv("ADDR")

func init() {
	switch RETURN_MODE {
	case "redirect":
		RETURN_MODE = "redirect"
	default:
		RETURN_MODE = "direct"
	}

	if CDN_PREFIX == "" {
		CDN_PREFIX = fmt.Sprintf("https://s3.amazonaws.com/%s/", S3_BUCKET)
	}

	if ADDR == "" {
		ADDR = "127.0.0.1:8000"
	}
}

func main() {
	stubHandler := &stubhandlers.StubHandler{
		ReturnMode: RETURN_MODE,
		CDNPrefix:  CDN_PREFIX,
		S3Bucket:   S3_BUCKET,
		S3Prefix:   S3_PREFIX,
	}

	mux := http.NewServeMux()
	mux.Handle("/", stubHandler)

	log.Fatal(http.ListenAndServe("127.0.0.1:8000", mux))
}
