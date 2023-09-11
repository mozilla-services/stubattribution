package dmglib

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestOpenFileInvalidFileName(t *testing.T) {
	if _, err := OpenFile("/path/to/invalid.dmg"); err == nil {
		t.Errorf("expected error")
	}
}

func TestOpenFile(t *testing.T) {
	file, err := OpenFile("../testdata/empty.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()
}

func TestParse(t *testing.T) {
	file, err := OpenFile("../testdata/empty.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if string(dmg.Koly.Signature[:]) != kolyBlockMagic {
		t.Errorf("unexpected koly block signature: %s, expected: %s", dmg.Koly.Signature, kolyBlockMagic)
	}
	if dmg.Koly.HeaderSize != kolyBlockSize {
		t.Errorf("unexpected koly block header size: %d, expected: %d", dmg.Koly.HeaderSize, kolyBlockSize)
	}
}

func TestClose(t *testing.T) {
	file := &DMGFile{}
	// Should not have any side effects.
	file.Close()
}

func TestParseDMGInvalidInputs(t *testing.T) {
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
		{input: makeValidInput(), expectedErr: ErrNoPropertyList},
		{input: makeInvalidInputWithPropertyList(), expectedErr: io.EOF},
	} {
		_, err := ParseDMG(strings.NewReader(tc.input))
		if err == nil {
			t.Errorf("expected error")
		}

		if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
			t.Errorf("expected error: %s, got: %s", tc.expectedErr, err)
		}

		if len(tc.expectedMsg) > 0 && !strings.Contains(err.Error(), tc.expectedMsg) {
			t.Errorf("expected error to contain: %s, got: %s", tc.expectedMsg, err)
		}
	}
}

func TestParseDMG(t *testing.T) {
	for _, tc := range []struct {
		testfile string
	}{
		{testfile: "../testdata/empty.dmg"},
		{testfile: "../testdata/attributable.dmg"},
	} {
		file, err := os.Open(tc.testfile)
		if err != nil {
			panic("couldn't open dmgfile")
		}

		data, err := ParseDMG(file)
		if err != nil {
			panic(err)
		}

		if string(data.Koly.Signature[:]) != kolyBlockMagic {
			t.Errorf("unexpected koly block signature: %s, expected: %s", data.Koly.Signature, kolyBlockMagic)
		}
		if data.Koly.HeaderSize != kolyBlockSize {
			t.Errorf("unexpected koly block header size: %d, expected: %d", data.Koly.HeaderSize, kolyBlockSize)
		}
	}
}
