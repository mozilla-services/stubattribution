package stubhandlers

import (
	"fmt"
	"net/http"

	"github.com/golang/groupcache/singleflight"
	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/pkg/errors"
)

// directHandler serves modified stub binaries directly
type directHandler struct {
	sfGroup *singleflight.Group

	BouncerBaseURL string
}

// NewDirectHandler returns a new direct type handler
func NewDirectHandler(bouncerBaseURL string) StubHandler {
	return &directHandler{
		sfGroup: new(singleflight.Group),
		BouncerBaseURL: bouncerBaseURL,
	}
}

// ServeStub serves stub bytes directly through handler
func (s *directHandler) ServeStub(w http.ResponseWriter, req *http.Request, code *attributioncode.Code) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := code.URLEncode()

	stub, err := sfFetchStub(s.sfGroup, bouncerURL(product, lang, os, s.BouncerBaseURL))
	if err != nil {
		return errors.Wrap(err, "fetchStub")
	}
	stub, err = modifyStub(stub, attributionCode)
	if err != nil {
		return err
	}

	// Cache response for one week
	w.Header().Set("Cache-Control", "max-age=604800")
	w.Header().Set("Content-Type", stub.contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(stub.body)))
	w.Write(stub.body)
	return nil
}
