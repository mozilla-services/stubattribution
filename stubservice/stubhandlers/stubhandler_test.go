package stubhandlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/quick"

	"github.com/mozilla-services/stubattribution/attributioncode"
	"github.com/mozilla-services/stubattribution/stubservice/backends"
)

func TestUniqueKey(t *testing.T) {
	f := func(url, code string) bool {
		key := uniqueKey(url, code)
		if len(key) != 64 {
			fmt.Errorf("key not 64 char url: %s, code %s: len: %d", url, code, len(key))
			return false
		}
		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestBouncerURL(t *testing.T) {
	url := bouncerURL("firefox", "en-US", "win")
	if url != "https://download.mozilla.org/?lang=en-US&os=win&product=firefox" {
		t.Errorf("url is not correct: %s", url)
	}
}

func TestRedirectResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/success":
			http.Redirect(w, req, "https://mozilla.org", 302)
		case "/nolocation":
			w.WriteHeader(302)
		case "/badstatus":
			w.WriteHeader(200)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("success", func(t *testing.T) {
		resp, err := redirectResponse(server.URL + "/success")
		if err != nil {
			t.Error(err)
		}
		if resp != "https://mozilla.org" {
			t.Errorf("Got %s", resp)
		}
	})

	t.Run("nolocation", func(t *testing.T) {
		_, err := redirectResponse(server.URL + "/nolocation")
		if !strings.Contains(err.Error(), "Location was empty") {
			t.Errorf("Incorrect error: %v", err)
		}
	})

	t.Run("badstatus", func(t *testing.T) {
		_, err := redirectResponse(server.URL + "/badstatus")
		if !strings.Contains(err.Error(), "returned 200, expecting 302") {
			t.Errorf("Incorrect error: %v", err)
		}
	})
}

func TestRedirectFull(t *testing.T) {
	testFileBytes, err := ioutil.ReadFile("../../testdata/test-stub.exe")
	if err != nil {
		t.Fatal("could not read test-stub.exe", err)
	}

	storage := backends.NewMapStorage()

	var server *httptest.Server
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/":
			http.Redirect(w, req, server.URL+"/pub/firefox/releases/51.0.1/win64/af/Firefox Setup 51.0.1.exe", 302)
			return
		case "/pub/firefox/releases/51.0.1/win64/af/Firefox Setup 51.0.1.exe":
			w.Write(testFileBytes)
			return
		}
		if strings.HasPrefix(req.URL.Path, "/cdn/") {
			item, ok := storage.Storage[strings.TrimPrefix(req.URL.Path, "/cdn/")]
			if !ok {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Type", item.ContentType)
			w.Write(item.Bytes)
			return
		}
	})
	server = httptest.NewServer(handler)
	defer server.Close()

	BouncerURL = server.URL
	defer func() {
		BouncerURL = "https://download.mozilla.org/"
	}()

	svc := NewStubService(
		NewRedirectHandler(storage, server.URL+"/cdn/", ""),
		&attributioncode.Validator{})

	recorder := httptest.NewRecorder()
	attributionCode := "campaign=%28not+set%29&content=%28not+set%29&medium=organic&source=www.google.com"
	base64Code := "Y2FtcGFpZ249KG5vdCtzZXQpJmNvbnRlbnQ9KG5vdCtzZXQpJm1lZGl1bT1vcmdhbmljJnNvdXJjZT13d3cuZ29vZ2xlLmNvbQ.."
	req := httptest.NewRequest("GET", `http://test/?product=firefox-stub&os=win&lang=en-US&attribution_code=`+url.QueryEscape(base64Code), nil)
	svc.ServeHTTP(recorder, req)

	if recorder.HeaderMap.Get("Location") == "" {
		t.Fatal("Location is not set")
	}

	resp, err := http.Get(recorder.HeaderMap.Get("Location"))
	if err != nil {
		t.Fatal("request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("request was not 200 res: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("could not read body", err)
	}

	if len(bodyBytes) != len(testFileBytes) {
		t.Error("Returned file was not the same length as the original file")
	}

	if !bytes.Contains(bodyBytes, []byte(url.QueryEscape(attributionCode))) {
		t.Error("Returned file did not contain attribution code")
	}
}

func TestDirectFull(t *testing.T) {
	testFileBytes, err := ioutil.ReadFile("../../testdata/test-stub.exe")
	if err != nil {
		t.Fatal("could not read test-stub.exe", err)
	}

	var server *httptest.Server
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/":
			http.Redirect(w, req, server.URL+"/pub/firefox/releases/51.0.1/win64/af/Firefox Setup 51.0.1.exe", 302)
			return
		case "/pub/firefox/releases/51.0.1/win64/af/Firefox Setup 51.0.1.exe":
			w.Write(testFileBytes)
			return
		}
	})
	server = httptest.NewServer(handler)
	defer server.Close()

	BouncerURL = server.URL
	defer func() {
		BouncerURL = "https://download.mozilla.org/"
	}()

	svc := NewStubService(
		NewDirectHandler(),
		&attributioncode.Validator{})

	recorder := httptest.NewRecorder()
	attributionCode := "campaign=%28not+set%29&content=%28not+set%29&medium=organic&source=www.google.com"
	base64Code := "Y2FtcGFpZ249KG5vdCtzZXQpJmNvbnRlbnQ9KG5vdCtzZXQpJm1lZGl1bT1vcmdhbmljJnNvdXJjZT13d3cuZ29vZ2xlLmNvbQ.."
	req := httptest.NewRequest("GET", `http://test/?product=firefox-stub&os=win&lang=en-US&attribution_code=`+url.QueryEscape(base64Code), nil)
	svc.ServeHTTP(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("request was not 200 res: %d", recorder.Code)
	}

	bodyBytes, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		t.Fatal("could not read body", err)
	}

	if len(bodyBytes) != len(testFileBytes) {
		t.Error("Returned file was not the same length as the original file")
	}

	if !bytes.Contains(bodyBytes, []byte(url.QueryEscape(attributionCode))) {
		t.Error("Returned file did not contain attribution code")
	}
}

func TestStubServiceErrorCases(t *testing.T) {
	svc := NewStubService(
		NewDirectHandler(),
		&attributioncode.Validator{})

	fetchURL := func(url string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", url, nil)
		svc.ServeHTTP(recorder, req)
		return recorder
	}

	t.Run("no attribution_code", func(t *testing.T) {
		recorder := fetchURL(`http://test/?product=firefox-stub&os=win&lang=en-US`)
		code := recorder.Code
		location := recorder.HeaderMap.Get("Location")
		if code != 302 || location != "https://download.mozilla.org/?lang=en-US&os=win&product=firefox-stub" {
			t.Errorf("service did not return bouncer redirect status: %d loc: %s", code, location)
		}
	})

	t.Run("invalid attribution_code", func(t *testing.T) {
		recorder := fetchURL(`http://test/?product=firefox-stub&os=win&lang=en-US&attribution_code=invalidcode`)
		code := recorder.Code
		location := recorder.HeaderMap.Get("Location")
		if code != 302 || location != "https://download.mozilla.org/?lang=en-US&os=win&product=firefox-stub" {
			t.Errorf("service did not return bouncer redirect status: %d loc: %s", code, location)
		}
	})
}
