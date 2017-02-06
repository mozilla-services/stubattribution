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
	if since, err := v.validateTimestamp(ts); err.Error() != "Timestamp: 2016-12-07 20:42:25 +0000 UTC is older than timeout: 10m0s" {
		if since < 1 {
			t.Errorf("Expected since > 0: since: %s", since)
		}
		t.Errorf("Expected error: %s", err)
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
			"c291cmNlPXd3dy5nb29nbGUuY29tJm1lZGl1bT1vcmdhbmljJmNhbXBhaWduPShub3Qgc2V0KSZjb250ZW50PShub3Qgc2V0KQ..", // source=www.google.com&medium=organic&campaign=(not set)&content=(not set)
			"campaign%3D%2528not%2Bset%2529%26content%3D%2528not%2Bset%2529%26medium%3Dorganic%26source%3Dwww.google.com",
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
			"c291cmNlPWdvb2dsZS5jb21tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbSZtZWRpdW09b3JnYW5pYyZjYW1wYWlnbj0obm90IHNldCkmY29udGVudD0obm90IHNldCk.", // source=google.commmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm&medium=organic&campaign=(not set)&content=(not set)
			"code longer than 200 characters",
		},
		{
			"bWVkaXVtPW9yZ2FuaWMmY2FtcGFpZ249KG5vdCBzZXQpJmNvbnRlbnQ9KG5vdCBzZXQp", // "medium=organic&campaign=(not set)&content=(not set)",
			"code is missing keys",
		},
		{
			"bm90YXJlYWxrZXk9b3JnYW5pYyZjYW1wYWlnbj0obm90IHNldCkmY29udGVudD0obm90IHNldCk.", // "notarealkey=organic&campaign=(not set)&content=(not set)",
			"notarealkey is not a valid attribution key",
		},
		{
			"c291cmNlPXd3dy5pbnZhbGlkZG9tYWluLmNvbSZtZWRpdW09b3JnYW5pYyZjYW1wYWlnbj0obm90IHNldCkmY29udGVudD0obm90IHNldCk.", // "source=www.invaliddomain.com&medium=organic&campaign=(not set)&content=(not set)",
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
