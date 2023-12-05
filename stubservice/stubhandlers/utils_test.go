package stubhandlers

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/mozilla-services/stubattribution/dmglib"
)

func TestFetchStub(t *testing.T) {
	t.Run("fetchStub", func(t *testing.T) {
		sampleBody := []byte("HelloWorld")
		// Create a server that returns a static JSON response
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(sampleBody)
		}))
		defer s.Close()
		got, err := fetchStub(s.URL)
		if err != nil {
			t.Errorf("fetchStub error: %s", err)
		}
		if want := string(sampleBody); want != string(got.body) {
			t.Errorf("Expected %s, got: %s", want, got)
		}
	})

	t.Run("fetchStub Get error", func(t *testing.T) {
		errMessage := `Get: Get "bogus://url": unsupported protocol scheme "bogus"`
		_, err := fetchStub("bogus://url")
		if err == nil {
			t.Error("expected an error with bogus URL!")
		}
		if err.Error() != errMessage {
			t.Errorf("Expected %s, got: %s", errMessage, err.Error())
		}
	})

	t.Run("fetchStub StatusCode", func(t *testing.T) {
		// Create a server that returns a static JSON response
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer s.Close()
		_, err := fetchStub(s.URL)
		if err == nil {
			t.Error("expected an error with invalid status code!")
		}
		if err.Error() != "invalid status code" {
			t.Errorf("Expected 'invalid status code', got: %s", err.Error())
		}
	})
}

const attributionChars = "abcdefghijklmnopqrstuvwxyz1234567890"

func _randomString(length int) string {
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

	st := &stub{
		body: fileBytes,
	}

	_, err = modifyStub(st, "hello=attribution&os=win", "win")
	if err != nil {
		t.Errorf("modifyStub error: %s", err)
	}
}

func TestModifyStubEXEFailWrite(t *testing.T) {
	fileBytes, err := os.ReadFile("../../testdata/test-stub.exe")
	if err != nil {
		t.Errorf("Error reading test EXE: %s", err)
	}

	st := &stub{
		body: fileBytes,
	}

	attribution := _randomString(100000)

	_, err = modifyStub(st, "ginormous="+attribution, "win")
	if err == nil {
		t.Errorf("expected an error, got: %s", reflect.TypeOf(err))
	}
}

func TestModifyStubDMG(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/attributable.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	st := &stub{
		body: dmg.Data,
	}

	_, err = modifyStub(st, "hello=attribution&os=osx", "osx")
	if err != nil {
		t.Errorf("modifyStub error: %s", err)
	}
}

func TestModifyStubFailDMGWrite(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/attributable.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	st := &stub{
		body: dmg.Data,
	}

	attribution := _randomString(100000)

	_, err = modifyStub(st, "ginormous="+attribution, "osx")
	if err == nil {
		t.Errorf("expected an error, got: %s", reflect.TypeOf(err))
	}
}

func TestModifyStubFailDMGBody(t *testing.T) {
	st := &stub{
		body: []byte("This is not a dmg!"),
	}
	_, err := modifyStub(st, "hello=errors", "osx")

	if err == nil {
		t.Errorf("expected an error, got: %s", reflect.TypeOf(err))
	}
}

func TestModifyStubFailOS(t *testing.T) {
	st := &stub{
		body: []byte("test"),
	}
	_, err := modifyStub(st, "hello=errors", "ardweeno")

	if err == nil {
		t.Errorf("expected an error, got: %s", reflect.TypeOf(err))
	}
}
