package stubhandlers

import (
	"testing"
	"time"
)

func TestParseTimeStamp(t *testing.T) {
	ts := "1481143345"
	res, err := parseTimeStamp(ts)
	if err != nil {
		t.Fatalf("Error parseTimeStamp: %v", err)
	}

	expected := time.Date(2016, time.December, 7, 20, 42, 25, 0, time.UTC)
	if !res.Equal(expected) {
		t.Errorf("expected: %v, res: %v", expected, res)
	}

}
