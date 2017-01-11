package attributioncode

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"testing"
	"testing/quick"
	"time"
)

func TestValidateTimeStamp(t *testing.T) {
	v := NewValidator("", 10*time.Minute)

	ts := "1481143345"
	if err := v.validateTimestamp(ts); err.Error() != "Timestamp is older than timeout: 10m0s" {
		t.Errorf("Expected error.", err)
	}
}

func TestValidateSignature(t *testing.T) {
	t.Run("static tests", func(t *testing.T) {
		v := NewValidator("testkey", 10*time.Minute)
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
			if err := v.validateSignature(testCase.Code, testCase.Sig); (err == nil) != testCase.Valid {
				t.Errorf("checking %s should equal: %v", testCase.Code, testCase.Valid)
			}
		}
	})

	t.Run("quick tests", func(t *testing.T) {
		f := func(code, key string) bool {
			v := NewValidator(key, 10*time.Minute)

			mac := hmac.New(sha256.New, []byte(key))
			mac.Write([]byte(code))
			if err := v.validateSignature(code, fmt.Sprintf("%x", mac.Sum(nil))); err != nil {
				t.Errorf("invalid signature: %v", err)
				return false
			}
			return true
		}
		if err := quick.Check(f, nil); err != nil {
			t.Errorf("failed: %v", err)
		}
	})
}

func TestValidateAttributionCode(t *testing.T) {
	v := &Validator{}

	validCodes := []struct {
		In  string
		Out string
	}{
		{
			"source%3Dwww.google.com%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"source%3Dwww.google.com%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
		},
	}
	for _, c := range validCodes {
		res, err := v.Validate(c.In, "")
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
		_, err := v.Validate(c.In, "")
		if err.Error() != c.Err {
			t.Errorf("err: %v != expected: %v", err, c.Err)
		}
	}
}
