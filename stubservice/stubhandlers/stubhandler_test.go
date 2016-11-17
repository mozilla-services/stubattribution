package stubhandlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
)

func TestValidateSignature(t *testing.T) {
	t.Run("static tests", func(t *testing.T) {
		service := &StubService{
			HMacKey: "testkey",
		}

		cases := []struct {
			Code  string
			Sig   string
			Valid bool
		}{
			{"testcode", "2608633175f9db16832c08342231423c2f9963396ca66f08350516a781ae8053", true},
			{"testcode", "2608633175f9db16832c08342231423c2f9963396ca66f08350516a781ae805Z", false},
			{"testcode", "2608633175f9db16832c08342231423c2f9963396ca66f08350516a781ae8052", false},
		}
		for _, testCase := range cases {
			if service.validateSignature(testCase.Code, testCase.Sig) != testCase.Valid {
				t.Errorf("checking %s should equal: %v", testCase.Code, testCase.Valid)
			}
		}
	})

	t.Run("quick tests", func(t *testing.T) {
		f := func(code, key string) bool {
			service := &StubService{
				HMacKey: key,
			}

			mac := hmac.New(sha256.New, []byte(key))
			mac.Write([]byte(code))
			return service.validateSignature(code, fmt.Sprintf("%x", mac.Sum(nil)))
		}
		if err := quick.Check(f, nil); err != nil {
			t.Errorf("failed: %v", err)
		}
	})
}

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

func TestValidateAttributionCode(t *testing.T) {
	validCodes := []struct {
		In  string
		Out string
	}{
		{
			"source%3Dwww.google.com%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"campaign=%28not+set%29&content=%28not+set%29&medium=organic&source=www.google.com",
		},
	}
	for _, c := range validCodes {
		res, err := validateAttributionCode(c.In)
		if err != nil {
			t.Errorf("err: %v, code: %s", err, c.In)
		}
		if res != c.Out {
			t.Errorf("res:%s != out:%s, code: %s", res, c.Out, c.In)
		}
	}

	invalidCodes := []struct {
		In  string
		Err string
	}{
		{
			"source%3Dgoogle.commmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"code longer than 200 characters",
		},
		{
			"medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"code is missing keys",
		},
		{
			"notarealkey%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"notarealkey is not a valid attribution key",
		},
		{
			"source%3Dwww.invaliddomain.com%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"source: www.invaliddomain.com is not in whitelist",
		},
	}
	for _, c := range invalidCodes {
		_, err := validateAttributionCode(c.In)
		if err.Error() != c.Err {
			t.Errorf("err: %v != expected: %v", err, c.Err)
		}
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
