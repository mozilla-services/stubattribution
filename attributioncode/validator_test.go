package attributioncode

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

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
		In                  string
		Out                 string
		RefererHeader       string
		ExpectedClientID    string
		ExpectedClientIDGA4 string
		ExpectedSessionID   string
	}{
		{
			// source=www.google.com&medium=organic&campaign=(not set)&content=(not set)
			In:                "c291cmNlPXd3dy5nb29nbGUuY29tJm1lZGl1bT1vcmdhbmljJmNhbXBhaWduPShub3Qgc2V0KSZjb250ZW50PShub3Qgc2V0KQ..",
			Out:               "campaign%3D%2528not%2Bset%2529%26content%3D%2528not%2Bset%2529%26dltoken%3D__DL_TOKEN__%26medium%3Dorganic%26source%3Dwww.google.com",
			RefererHeader:     "",
			ExpectedClientID:  "",
			ExpectedSessionID: "",
		},
		{
			// source=www.google.com&medium=organic&campaign=(not set)
			In:                "c291cmNlPXd3dy5nb29nbGUuY29tJm1lZGl1bT1vcmdhbmljJmNhbXBhaWduPShub3Qgc2V0KQ..",
			Out:               "campaign%3D%2528not%2Bset%2529%26content%3D%2528not%2Bset%2529%26dltoken%3D__DL_TOKEN__%26medium%3Dorganic%26source%3Dwww.google.com",
			RefererHeader:     "",
			ExpectedClientID:  "",
			ExpectedSessionID: "",
		},
		{
			// source=www.google.com&medium=organic&campaign=(not set)&content=(not set)&variation=f1&experiment=e1
			In:                "c291cmNlPXd3dy5nb29nbGUuY29tJm1lZGl1bT1vcmdhbmljJmNhbXBhaWduPShub3Qgc2V0KSZjb250ZW50PShub3Qgc2V0KSZ2YXJpYXRpb249ZjEmZXhwZXJpbWVudD1lMQ..",
			Out:               "campaign%3D%2528not%2Bset%2529%26content%3D%2528not%2Bset%2529%26dltoken%3D__DL_TOKEN__%26experiment%3De1%26medium%3Dorganic%26source%3Dwww.google.com%26variation%3Df1",
			RefererHeader:     "",
			ExpectedClientID:  "",
			ExpectedSessionID: "",
		},
		{
			// source=addons.mozilla.org&medium=referral&campaign=amo-fx-cta-3006&content=rta:e2I5ZGIxNmE0LTZlZGMtNDdlYy1hMWY0LWI4NjI5MmVkMjExZH0&experiment=(not set)&variation=(not set)&ua=edge&visit_id=(not set)
			In:                "c291cmNlPWFkZG9ucy5tb3ppbGxhLm9yZyZtZWRpdW09cmVmZXJyYWwmY2FtcGFpZ249YW1vLWZ4LWN0YS0zMDA2JmNvbnRlbnQ9cnRhOmUySTVaR0l4Tm1FMExUWmxaR010TkRkbFl5MWhNV1kwTFdJNE5qSTVNbVZrTWpFeFpIMCZleHBlcmltZW50PShub3Qgc2V0KSZ2YXJpYXRpb249KG5vdCBzZXQpJnVhPWVkZ2UmdmlzaXRfaWQ9KG5vdCBzZXQp",
			Out:               "campaign%3Damo-fx-cta-3006%26content%3Drta%253Ae2I5ZGIxNmE0LTZlZGMtNDdlYy1hMWY0LWI4NjI5MmVkMjExZH0%26dltoken%3D__DL_TOKEN__%26experiment%3D%2528not%2Bset%2529%26medium%3Dreferral%26source%3Daddons.mozilla.org%26ua%3Dedge%26variation%3D%2528not%2Bset%2529",
			RefererHeader:     "https://www.mozilla.org/",
			ExpectedClientID:  "(not set)",
			ExpectedSessionID: "",
		},
		{
			// source=addons.mozilla.org&medium=referral&campaign=amo-fx-cta-3006&content=rta:e2I5ZGIxNmE0LTZlZGMtNDdlYy1hMWY0LWI4NjI5MmVkMjExZH0&experiment=(not set)&variation=(not set)&ua=edge&visit_id=(not set)
			In:                "c291cmNlPWFkZG9ucy5tb3ppbGxhLm9yZyZtZWRpdW09cmVmZXJyYWwmY2FtcGFpZ249YW1vLWZ4LWN0YS0zMDA2JmNvbnRlbnQ9cnRhOmUySTVaR0l4Tm1FMExUWmxaR010TkRkbFl5MWhNV1kwTFdJNE5qSTVNbVZrTWpFeFpIMCZleHBlcmltZW50PShub3Qgc2V0KSZ2YXJpYXRpb249KG5vdCBzZXQpJnVhPWVkZ2UmdmlzaXRfaWQ9KG5vdCBzZXQp",
			Out:               "campaign%3Damo-fx-cta-3006%26content%3Drta%253Ae2I5ZGIxNmE0LTZlZGMtNDdlYy1hMWY0LWI4NjI5MmVkMjExZH0%26dltoken%3D__DL_TOKEN__%26experiment%3D%2528not%2Bset%2529%26medium%3Dreferral%26source%3Daddons.mozilla.org%26ua%3Dedge%26variation%3D%2528not%2Bset%2529",
			RefererHeader:     "https://www.mozilla.org/test/other/paths",
			ExpectedClientID:  "(not set)",
			ExpectedSessionID: "",
		},
		{
			// campaign=testcampaign&content=testcontent&experiment=exp1&medium=testmedium&source=mozilla.com&timestamp=1670358814&variation=var1&visit_id=vid
			In:            "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNvbnRlbnQ9dGVzdGNvbnRlbnQmZXhwZXJpbWVudD1leHAxJm1lZGl1bT10ZXN0bWVkaXVtJnNvdXJjZT1tb3ppbGxhLmNvbSZ0aW1lc3RhbXA9MTY3MDM1ODgxNCZ2YXJpYXRpb249dmFyMSZ2aXNpdF9pZD12aWQ.",
			Out:           "campaign%3Dtestcampaign%26content%3Dtestcontent%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader: "",
			// Only `visit_id` is passed here.
			ExpectedClientID:  "vid",
			ExpectedSessionID: "",
		},
		{
			// campaign=testcampaign&content=testcontent&experiment=exp1&medium=testmedium&source=mozilla.com&timestamp=1670358814&variation=var1&visit_id=vid&session_id=sid
			In:            "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNvbnRlbnQ9dGVzdGNvbnRlbnQmZXhwZXJpbWVudD1leHAxJm1lZGl1bT10ZXN0bWVkaXVtJnNvdXJjZT1tb3ppbGxhLmNvbSZ0aW1lc3RhbXA9MTY3MDM1ODgxNCZ2YXJpYXRpb249dmFyMSZ2aXNpdF9pZD12aWQmc2Vzc2lvbl9pZD1zaWQ.",
			Out:           "campaign%3Dtestcampaign%26content%3Dtestcontent%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader: "",
			// `visit_id` is present, `client_id` isn't.
			ExpectedClientID:  "vid",
			ExpectedSessionID: "sid",
		},
		{
			// campaign=testcampaign&client_id=cid&content=testcontent&experiment=exp1&medium=testmedium&session_id=sid&source=mozilla.com&timestamp=1677165697&variation=var1
			In:            "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNsaWVudF9pZD1jaWQmY29udGVudD10ZXN0Y29udGVudCZleHBlcmltZW50PWV4cDEmbWVkaXVtPXRlc3RtZWRpdW0mc2Vzc2lvbl9pZD1zaWQmc291cmNlPW1vemlsbGEuY29tJnRpbWVzdGFtcD0xNjc3MTY1Njk3JnZhcmlhdGlvbj12YXIx",
			Out:           "campaign%3Dtestcampaign%26content%3Dtestcontent%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader: "",
			// `client_id` is present, `visit_id` isn't.
			ExpectedClientID:  "cid",
			ExpectedSessionID: "sid",
		},
		{
			// campaign=testcampaign&client_id=cid&content=testcontent&experiment=exp1&medium=testmedium&session_id=sid&source=mozilla.com&timestamp=1677166561&variation=var1&visit_id=vid
			In:            "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNsaWVudF9pZD1jaWQmY29udGVudD10ZXN0Y29udGVudCZleHBlcmltZW50PWV4cDEmbWVkaXVtPXRlc3RtZWRpdW0mc2Vzc2lvbl9pZD1zaWQmc291cmNlPW1vemlsbGEuY29tJnRpbWVzdGFtcD0xNjc3MTY2NTYxJnZhcmlhdGlvbj12YXIxJnZpc2l0X2lkPXZpZA..",
			Out:           "campaign%3Dtestcampaign%26content%3Dtestcontent%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader: "",
			// Both `client_id` and `visit_id` are passed. In this case, `client_id`,
			// which is non-empty, is preferred.
			ExpectedClientID:  "cid",
			ExpectedSessionID: "sid",
		},
		{
			// campaign=testcampaign&client_id=&content=testcontent&experiment=exp1&medium=testmedium&session_id=sid&source=mozilla.com&timestamp=1677166718&variation=var1&visit_id=vid
			In:            "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNsaWVudF9pZD0mY29udGVudD10ZXN0Y29udGVudCZleHBlcmltZW50PWV4cDEmbWVkaXVtPXRlc3RtZWRpdW0mc2Vzc2lvbl9pZD1zaWQmc291cmNlPW1vemlsbGEuY29tJnRpbWVzdGFtcD0xNjc3MTY2NzE4JnZhcmlhdGlvbj12YXIxJnZpc2l0X2lkPXZpZA..",
			Out:           "campaign%3Dtestcampaign%26content%3Dtestcontent%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader: "",
			// Both `client_id` and `visit_id` are passed but `client_id` is an empty
			// string so we prefer `visit_id`.
			ExpectedClientID:  "vid",
			ExpectedSessionID: "sid",
		},
		{
			// campaign=testcampaign&content=testcontent&experiment=exp1&medium=testmedium&source=mozilla.com&timestamp=1670358814&variation=var1&dlsource=mozorg
			In:                "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNvbnRlbnQ9dGVzdGNvbnRlbnQmZXhwZXJpbWVudD1leHAxJm1lZGl1bT10ZXN0bWVkaXVtJnNvdXJjZT1tb3ppbGxhLmNvbSZ0aW1lc3RhbXA9MTY3MDM1ODgxNCZ2YXJpYXRpb249dmFyMSZkbHNvdXJjZT1tb3pvcmc.",
			Out:               "campaign%3Dtestcampaign%26content%3Dtestcontent%26dlsource%3Dmozorg%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader:     "",
			ExpectedClientID:  "",
			ExpectedSessionID: "",
		},
		{
			// campaign=testcampaign&client_id=cid&client_id_ga4=cid_ga4&content=testcontent&experiment=exp1&medium=testmedium&session_id=sid&source=mozilla.com&timestamp=1707731136&variation=var1
			In:                  "Y2FtcGFpZ249dGVzdGNhbXBhaWduJmNsaWVudF9pZD1jaWQmY2xpZW50X2lkX2dhND1jaWRfZ2E0JmNvbnRlbnQ9dGVzdGNvbnRlbnQmZXhwZXJpbWVudD1leHAxJm1lZGl1bT10ZXN0bWVkaXVtJnNlc3Npb25faWQ9c2lkJnNvdXJjZT1tb3ppbGxhLmNvbSZ0aW1lc3RhbXA9MTcwNzczMTA3OSZ2YXJpYXRpb249dmFyMQ..",
			Out:                 "campaign%3Dtestcampaign%26content%3Dtestcontent%26dltoken%3D__DL_TOKEN__%26experiment%3Dexp1%26medium%3Dtestmedium%26source%3Dmozilla.com%26variation%3Dvar1",
			RefererHeader:       "",
			ExpectedClientID:    "cid",
			ExpectedSessionID:   "sid",
			ExpectedClientIDGA4: "cid_ga4",
		},
	}
	for _, c := range validCodes {
		code, err := v.Validate(c.In, "", c.RefererHeader)
		if err != nil {
			t.Errorf("err: %v, code: %s", err, c.In)
		}

		res := code.URLEncode()
		if res != strings.ReplaceAll(c.Out, "__DL_TOKEN__", code.DownloadToken()) {
			t.Errorf("res:%s != out:%s, code: %s", res, c.Out, c.In)
		}

		if c.ExpectedClientID != code.ClientID {
			t.Errorf("Expected ClientID: '%s', got: '%s', code: %s", c.ExpectedClientID, code.ClientID, c.In)
		}

		if c.ExpectedClientIDGA4 != code.ClientIDGA4 {
			t.Errorf("Expected ClientIDGA4: '%s', got: '%s', code: %s", c.ExpectedClientIDGA4, code.ClientIDGA4, c.In)
		}

		if c.ExpectedSessionID != code.SessionID {
			t.Errorf("Expected SessionID: '%s', got: '%s', code: %s", c.ExpectedSessionID, code.SessionID, c.In)
		}
	}

	invalidCodes := []struct {
		In            string
		Err           string
		Sig           string
		RefererHeader string
	}{
		{
			"YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=",
			"base64 code longer than 5000 characters",
			"",
			"",
		},
		{
			"c291cmNlPWdvb2dsZS5jb21tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tbW1tJm1lZGl1bT1vcmdhbmljJmNhbXBhaWduPShub3Qgc2V0KSZjb250ZW50PShub3Qgc2V0KQ..", // source=google.commmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm&medium=organic&campaign=(not set)&content=(not set)
			"code longer than 1010 characters",
			"",
			"",
		},
		{
			"test",
			"sig longer than 5000 characters",
			strings.Repeat("s", 5001),
			"",
		},
		{
			"bm90YXJlYWxrZXk9b3JnYW5pYyZjYW1wYWlnbj0obm90IHNldCkmY29udGVudD0obm90IHNldCk.", // "notarealkey=organic&campaign=(not set)&content=(not set)",
			"notarealkey is not a valid attribution key",
			"",
			"",
		},
		{
			"", // blank
			"code is empty",
			"",
			"",
		},
		{
			"c291cmNlPWFkZG9ucy5tb3ppbGxhLm9yZyZtZWRpdW09cmVmZXJyYWwmY2FtcGFpZ249YW1vLWZ4LWN0YS0zMDA2JmNvbnRlbnQ9cnRhOmUySTVaR0l4Tm1FMExUWmxaR010TkRkbFl5MWhNV1kwTFdJNE5qSTVNbVZrTWpFeFpIMCZleHBlcmltZW50PShub3Qgc2V0KSZ2YXJpYXRpb249KG5vdCBzZXQpJnVhPWVkZ2UmdmlzaXRfaWQ9KG5vdCBzZXQp", // source=addons.mozilla.org&medium=referral&campaign=amo-fx-cta-3006&content=rta:e2I5ZGIxNmE0LTZlZGMtNDdlYy1hMWY0LWI4NjI5MmVkMjExZH0&experiment=(not set)&variation=(not set)&ua=edge&visit_id=(not set)
			"RTAMO attribution does not have https://www.mozilla.org referer header",
			"",
			"https://invalid-referer.fake",
		},
	}
	for _, c := range invalidCodes {
		_, err := v.Validate(c.In, c.Sig, c.RefererHeader)
		if err == nil {
			t.Errorf("err was nil, expected: %v", c.Err)
			continue
		}
		if err.Error() != c.Err {
			t.Errorf("err: %v != expected: %v", err, c.Err)
		}
	}
}

func TestFromRTAMO(t *testing.T) {
	invalidCodes := []string{" rta:123", "wrongcode", "rta"}
	validCodes := []string{"rta:123", "rta:abc"}

	for _, v := range invalidCodes {
		c := Code{Content: v}

		if c.FromRTAMO() {
			t.Errorf("Invalid code matched regex: %s", v)
		}
	}

	for _, v := range validCodes {
		c := Code{Content: v}

		if !c.FromRTAMO() {
			t.Errorf("Valid code did not match regex: %s", v)
		}
	}
}
