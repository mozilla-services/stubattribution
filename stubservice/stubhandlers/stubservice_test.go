package stubhandlers

import (
	"testing"
	"testing/quick"
)

func TestParseQueryNoEscape(t *testing.T) {
	query, _ := parseQueryNoEscape("product=test-stub&os=win&lang=en-US&attribution_code=source%3Dgoogle%26medium%3Dpaidsearch%26campaign%3Dfoopy%26content%3D%28not+set%29%26timestamp%3D1482358230")
	if code := query.Get("attribution_code"); code != "source%3Dgoogle%26medium%3Dpaidsearch%26campaign%3Dfoopy%26content%3D%28not+set%29%26timestamp%3D1482358230" {
		t.Errorf("unexpected attribution_code: %s", code)
	}
}

func TestTrimToLen(t *testing.T) {
	f := func(s string, l int) bool {
		// make sure l is positive
		if l < 0 {
			l = l * -1
		}

		res := trimToLen(s, l)
		return len(res) <= l
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
