package dmgmodify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/mozilla-services/stubattribution/dmglib"
)

func TestWriteAttributionCode(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/attributable.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	newCode := []byte("updated attribution code")

	err = WriteAttributionCode(dmg, newCode)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Ensure the updated dmg contains the new attribution code
	if !bytes.Contains(dmg.Data, newCode) {
		t.Error("updated dmg data does not contain new attribution code!")
	}

	// Ensure the updated dmg can be parsed
	newDmg, err := dmglib.ParseDMG(bytes.NewReader(dmg.Data))
	if err != nil {
		t.Errorf("updated dmg data cannot be parsed, got error: %s", err)
	}

	// Verify that the newly parsed data matches the modified in-memory data.
	if !reflect.DeepEqual(dmg, newDmg) {
		t.Errorf("updated dmg is not the same as its source after being reparsed!")
	}

	// Compare the hash against what we expect it to be
	f, err := os.Open("../../testdata/attributed.dmg")
	expectedHash := sha256.New()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	_, err = io.Copy(expectedHash, f)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	expectedHex := hex.EncodeToString(expectedHash.Sum(nil))

	newHash := sha256.Sum256(newDmg.Data)
	newHex := hex.EncodeToString(newHash[:])

	if expectedHex != newHex {
		t.Errorf("wrong hash for updated dmg: %s, expected: %s", newHex, expectedHex)
	}
}

func TestWriteAttributionCodeTooLong(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/attributable.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	newCode := bytes.Repeat([]byte("Z"), 2000)

	err = WriteAttributionCode(dmg, newCode)

	if err != ErrCodeTooLong {
		t.Errorf("expected ErrCodeTooLong, got: %s", err)
	}
}

func TestWriteAttributionCodeSentinelMissing(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/empty.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	newCode := []byte("updated attribution code")

	err = WriteAttributionCode(dmg, newCode)

	if err != ErrSentinelMissing {
		t.Errorf("expected ErrSentinelMissing, got: %s", err)
	}
}
