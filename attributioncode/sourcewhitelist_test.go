package attributioncode

import "testing"

func TestIsWhitelisted(t *testing.T) {
	cases := []struct {
		Domain string
		Valid  bool
	}{
		{"www-demo4.allizom.org", true},
		{"google", true},
		{"www.randomdomain.com", false},
	}
	for _, c := range cases {
		if isWhitelisted(c.Domain) != c.Valid {
			t.Errorf("Domain: %s result: %v expected: %v", c.Domain, !c.Valid, c.Valid)
		}
	}
}
