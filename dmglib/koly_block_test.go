package dmglib

import (
	"strings"
	"testing"
)

func TestParseKolyBlockWithInvalidInputs(t *testing.T) {
	for _, tc := range []struct {
		input       string
		expectedMsg string
		expectedErr error
	}{
		{input: "", expectedMsg: "Seek"},
		{input: strings.Repeat("A", 511), expectedMsg: "Seek"},
		{input: strings.Repeat("A", 512), expectedErr: ErrNotKolyBlock},
		// Block starts with the right magic value but size is 0
		{input: makeInput(0), expectedErr: ErrInvalidHeaderSize},
		// Block starts with the right magic value but size is 511
		{input: makeInput(511), expectedErr: ErrInvalidHeaderSize},
	} {
		_, err := parseKolyBlock(strings.NewReader(tc.input))
		if err == nil {
			t.Errorf("expected error")
		}

		if tc.expectedErr != nil && err != tc.expectedErr {
			t.Errorf("expected error: %s, got: %s", tc.expectedErr, err)
		}

		if len(tc.expectedMsg) > 0 && !strings.Contains(err.Error(), tc.expectedMsg) {
			t.Errorf("expected error to contain: %s, got: %s", tc.expectedMsg, err)
		}
	}
}

func TestParseKolyBlock(t *testing.T) {
	block, err := parseKolyBlock(strings.NewReader(makeValidInput()))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if block.HeaderSize != kolyBlockSize {
		t.Errorf("expected header size: %d, got: %d", kolyBlockSize, block.HeaderSize)
	}
}
