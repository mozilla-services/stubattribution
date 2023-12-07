package stubhandlers

import (
	"bytes"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mozilla-services/stubattribution/dmglib"
)

const attributionChars = "abcdefghijklmnopqrstuvwxyz1234567890"

func TestFetchStub(t *testing.T) {
	t.Run("fetchStub", func(t *testing.T) {
		// Empty json object
		sampleBody := []byte(`{"hello": "world"}`)
		// Create a server that returns a static JSON response
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(sampleBody)
		}))
		defer s.Close()
		got, err := fetchStub(s.URL)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !bytes.Equal(got.body, sampleBody) {
			t.Errorf("Expected %s, got: %s", sampleBody, got.body)
		}
	})

	t.Run("fetchStub - invalid URL", func(t *testing.T) {
		errMessage := `Get: Get "bogus://url": unsupported protocol scheme "bogus"`
		_, err := fetchStub("bogus://url")
		if err == nil {
			t.Error("Expected an error with bogus URL")
		}
		if err.Error() != errMessage {
			t.Errorf("Expected %s, got: %s", errMessage, err.Error())
		}
	})

	t.Run("fetchStub - non-OK status code", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer s.Close()
		_, err := fetchStub(s.URL)
		if err == nil {
			t.Error("Expected an error with invalid status code")
		}
		if err.Error() != "invalid status code" {
			t.Errorf("Expected 'invalid status code', got: %s", err.Error())
		}
	})
}

func makeRandomString(length int) string {
	str := make([]byte, length)
	for i := range str {
		str[i] = attributionChars[rand.Intn(len(attributionChars))]
	}
	return string(str)
}

func TestModifyStubEXE(t *testing.T) {
	fileBytes, err := os.ReadFile("../../testdata/test-stub.exe")
	if err != nil {
		t.Errorf("Error reading test EXE: %s", err)
	}

	t.Run("modifyStub - EXE success", func(t *testing.T) {
		st := &stub{
			body: fileBytes,
		}
		_, err = modifyStub(st, "hello=attribution&os=win", "win")
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
	})

	t.Run("modifyStub - EXE fail", func(t *testing.T) {
		st := &stub{
			body: fileBytes,
		}
		attribution := makeRandomString(100000)

		_, err = modifyStub(st, "ginormous="+attribution, "win")
		if err == nil {
			t.Error("Expected an error writing a huge attribution code")
		}
	})
}

func TestModifyStubDMG(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/attributable.dmg")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	t.Run("modifyStub - DMG success", func(t *testing.T) {
		st := &stub{
			body: dmg.Data,
		}

		_, err = modifyStub(st, "hello=attribution&os=osx", "osx")
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
	})

	t.Run("modifyStub - DMG parse failure", func(t *testing.T) {
		st := &stub{
			body: []byte("This is not a dmg!"),
		}
		_, err := modifyStub(st, "hello=errors", "osx")

		if err == nil {
			t.Error("Expected an error failing to parse")
		}
	})

	t.Run("modifyStub - DMG writing failure", func(t *testing.T) {
		st := &stub{
			body: dmg.Data,
		}
		attribution := makeRandomString(100000)

		_, err = modifyStub(st, "ginormous="+attribution, "osx")
		if err == nil {
			t.Error("Expected an error writing a huge attribution code")
		}
	})
}

func TestModifyStubFailOS(t *testing.T) {
	st := &stub{
		body: []byte("test"),
	}
	_, err := modifyStub(st, "hello=errors", "ardweeno")

	if err == nil {
		t.Error("Expected an error for unsupported OS")
	}
}
