package stubhandlers

import (
	"net/http"

	"github.com/mozilla-services/stubattribution/attributioncode"
)

// StubHandler interface returns an error if anything went wrong
// else it handled the request successfully
type StubHandler interface {
	ServeStub(http.ResponseWriter, *http.Request, *attributioncode.Code) error
}
