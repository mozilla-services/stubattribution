package dmgreader

import (
	"errors"
	"io"
	"strings"
	"testing"

	"howett.net/plist"
)

func TestOpenFileInvalidFileName(t *testing.T) {
	if _, err := OpenFile("/path/to/invalid.dmg"); err == nil {
		t.Errorf("expected error")
	}
}

func TestOpenFile(t *testing.T) {
	file, err := OpenFile("tests/fixtures/empty.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()
}

func TestClose(t *testing.T) {
	file := &DMG{}
	// Should not have any side effects.
	file.Close()
}

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

func TestParseXMLPropertyListInvalidInputs(t *testing.T) {
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
		_, err := parseXMLPropertyList(strings.NewReader(tc.input))
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

func TestParseXMLPropertyList(t *testing.T) {
	// Some fake data, coming from the plist library.
	plist := struct {
		InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
		BandSize              uint64 `plist:"band-size"`
	}{
		InfoDictionaryVersion: "6.0",
		BandSize:              8388608,
	}

	data, err := parseXMLPropertyList(strings.NewReader(makeInputWithPropertyList(plist)))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if data["CFBundleInfoDictionaryVersion"] != "6.0" {
		t.Errorf("unexpected CFBundleInfoDictionaryVersion value: %s", data["CFBundleInfoDictionaryVersion"])
	}

	if data["band-size"] != uint64(8388608) {
		t.Errorf("unexpected band-size value: %d", data["band-size"])
	}
}

func makeInput(headerSize uint32) string {
	var sb strings.Builder

	block := newKolyBlock()
	block.HeaderSize = headerSize
	block.write(&sb)

	return sb.String()
}

func makeValidInput() string {
	return makeInput(kolyBlockSize)
}

func makeInvalidInputWithPropertyList() string {
	var sb strings.Builder

	// Add some padding at the beginning.
	for i := 0; i < 10; i++ {
		sb.WriteByte(0)
	}

	block := newKolyBlock()
	block.Version = 4
	block.XMLOffset = 2
	block.XMLLength = 1000 // This is clearly invalid.
	block.write(&sb)

	return sb.String()
}

func makeInputWithPropertyList(data interface{}) string {
	var sb strings.Builder

	// Write some padding first...
	offset := 2
	for i := 0; i < offset; i++ {
		sb.WriteByte(0)
	}
	// Then write the XML property list...
	err := plist.NewEncoder(&sb).Encode(data)
	if err != nil {
		panic(err)
	}
	dataLength := sb.Len() - offset
	// Some some padding...
	for i := 0; i < 5; i++ {
		sb.WriteByte(0)
	}
	// And finally the koly block.
	block := newKolyBlock()
	block.Version = 4
	block.XMLOffset = uint64(offset)
	block.XMLLength = uint64(dataLength)
	block.write(&sb)

	return sb.String()
}
