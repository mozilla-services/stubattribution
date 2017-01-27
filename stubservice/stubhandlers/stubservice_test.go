package stubhandlers

import (
	"testing"
	"testing/quick"
)

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
