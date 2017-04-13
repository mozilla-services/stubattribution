package attributioncode

import "testing"

func TestIsWhitelisted(t *testing.T) {
	cases := []struct {
		Domain string
		Valid  bool
	}{
		{"www-demo4.allizom.org", true},
		{"google", true},
		{"foo.search.yahoo.com", true},
		{"www.google.com.mx", true},
		{"www.google.co.id", true},
		{"www.google.cz", true},
		{"www.randomdomain.com", false},
	}
	for _, c := range cases {
		if IsWhitelisted(c.Domain) != c.Valid {
			t.Errorf("Domain: %s result: %v expected: %v", c.Domain, !c.Valid, c.Valid)
		}
	}
}
