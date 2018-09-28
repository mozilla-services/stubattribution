package attributioncode

import "testing"

func TestIsWhitelisted(t *testing.T) {
	cases := []struct {
		Domain string
		Valid  bool
	}{
		{"bedrock-demo-bouncer-stage.us-west.moz.works", true},
		{"firefox.com", true},
		{"www.firefox.com", true},
		{"screenshots.firefox.com", true},
		{"testpilot.firefox.com", true},
		{"www-demo4.allizom.org", true},
		{"google", true},
		{"foo.search.yahoo.com", true},
		{"www.google.com.mx", true},
		{"www.google.co.id", true},
		{"www.google.cz", true},
		{"www.randomdomain.com", false},

		{"addons.mozilla.org", true},
		{"developer.mozilla.org", true},
		{"mozilla.org", true},
		{"support.mozilla.org", true},
		{"www.mozilla.org", true},
	}
	for _, c := range cases {
		if isWhitelisted(c.Domain) != c.Valid {
			t.Errorf("Domain: %s result: %v expected: %v", c.Domain, !c.Valid, c.Valid)
		}
	}
}
