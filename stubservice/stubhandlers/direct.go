package stubhandlers

import (
	"fmt"
	"net/http"

	"github.com/golang/groupcache/singleflight"
	"github.com/pkg/errors"
)

// directHandler serves modified stub binaries directly
type directHandler struct {
	sfGroup *singleflight.Group
}

// NewDirectHandler returns a new direct type handler
func NewDirectHandler() StubHandler {
	return &directHandler{
		sfGroup: new(singleflight.Group),
	}
}

// ServeStub serves stub bytes directly through handler
func (s *directHandler) ServeStub(w http.ResponseWriter, req *http.Request, code string) error {
	query := req.URL.Query()
	product := query.Get("product")
	lang := query.Get("lang")
	os := query.Get("os")
	attributionCode := code

	stub, err := sfFetchStub(s.sfGroup, bouncerURL(product, lang, os))
	if err != nil {
		return errors.Wrap(err, "fetchStub")
	}
	stub, err = modifyStub(stub, attributionCode)
	if err != nil {
		return errors.Wrap(err, "modifyStub")
	}

	// Cache response for one week
	w.Header().Set("Cache-Control", "max-age=604800")
	w.Header().Set("Content-Type", stub.contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(stub.body)))
	w.Write(stub.body)
	return nil
}
