package stubhandlers

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// directHandler serves modified stub binaries directly
type directHandler struct {
}

// NewDirectHandler returns a new direct type handler
func NewDirectHandler() StubHandler {
	return &directHandler{}
}

// ServeStub serves stub bytes directly through handler
func (s *directHandler) ServeStub(w http.ResponseWriter, req *http.Request, code string) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := code

	stub, err := fetchModifyStub(bouncerURL(product, lang, os), attributionCode)
	if err != nil {
		return errors.Wrap(err, "fetchModifyStub")
	}

	// Cache response for one week
	w.Header().Set("Cache-Control", "max-age=604800")
	w.Header().Set("Content-Type", stub.Resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(stub.Data)))
	w.Write(stub.Data)
	return nil
}
