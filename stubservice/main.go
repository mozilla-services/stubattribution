package main

import (
	"log"
	"net/http"

	"github.com/mozilla-services/go-stubattribution/stubservice/stubhandlers"
)

func main() {

	log.Fatal(http.ListenAndServe("127.0.0.1:8000", http.HandlerFunc(stubhandlers.StubHandler)))
}
