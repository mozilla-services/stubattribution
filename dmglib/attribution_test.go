package dmglib

import (
	"errors"
	"testing"
)

// This is an encoded version of an AttributionResource. Note: in this form all of the fields are
// endian flipped.
var expectedAttributionData = AttributionResource{
	Signature:                  0x61747472,
	Version:                    1,
	BeforeCompressedChecksum:   3552061997,
	BeforeCompressedLength:     1070,
	BeforeUncompressedChecksum: 3115706219,
	BeforeUncompressedLength:   3407872,
	RawPos:                     1070,
	RawLength:                  524288,
	RawChecksum:                2803208855,
	AfterCompressedChecksum:    1531427171,
	AfterCompressedLength:      149233958,
	AfterUncompressedChecksum:  4111218357,
	AfterUncompressedLength:    468975616,
}

var emptyAttributionData = AttributionResource{
	Signature:                  0,
	Version:                    0,
	BeforeCompressedChecksum:   0,
	BeforeCompressedLength:     0,
	BeforeUncompressedChecksum: 0,
	BeforeUncompressedLength:   0,
	RawPos:                     0,
	RawLength:                  0,
	RawChecksum:                0,
	AfterCompressedChecksum:    0,
	AfterCompressedLength:      0,
	AfterUncompressedChecksum:  0,
	AfterUncompressedLength:    0,
}

func TestParseAttribution(t *testing.T) {
	for rawData, attrData := range map[string]AttributionResource{
		"cnR0YQEAAAAtKrjTLgQAAAAAAABr57W5AAA0AAAAAAAuBAAAAAAAAAAACAAAAAAAl5IVp2O5R1smIeUIAAAAALU2DPUAAPQbAAAAAA==":             expectedAttributionData,
		"\tcnR0YQEAAAAtKrjTL\t\t\tgQAAAAAAAB\nr57W5AAA0AAAAAAAuBAAAAAAAAAAACAAAAAAAl5IVp2O5R1smIeUIAAAAALU2DPUAAPQbAAAAAA\n==": expectedAttributionData,
		"": emptyAttributionData,
	} {
		res, err := ParseAttribution(rawData)
		if err != nil {
			t.Errorf("unexpected error: %s, data was: %s", err, rawData)
		}

		if *res != attrData {
			t.Errorf("attribution data not parsed correctly, expected %+v, got %+v, data was: %s", attrData, res, rawData)
		}
	}
}

func TestParseEmptyAttribution(t *testing.T) {
	res, err := ParseAttribution("")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if *res != emptyAttributionData {
	}
}

func TestParseAttributionInvalid(t *testing.T) {
	for _, tc := range []struct {
		input       string
		expectedErr error
	}{
		{input: "abcdefgh123445678", expectedErr: ErrBadAttrBase64},
		{input: "cnR0YQIAAAAtKrjTLgQAAAAAAABr57W5AAA0AAAAAAAuBAAAAAAAAAAACg==", expectedErr: ErrBadAttrLength},
		// Same as encodedAttributionData, except signature is zzzz
		{input: "enp6egEAAAAtKrjTLgQAAAAAAABr57W5AAA0AAAAAAAuBAAAAAAAAAAACAAAAAAAl5IVp2O5R1smIeUIAAAAALU2DPUAAPQbAAAAAA==", expectedErr: ErrBadAttrSignature},
		// Same as encodedAttributionData, except version is set to 2
		{input: "cnR0YQIAAAAtKrjTLgQAAAAAAABr57W5AAA0AAAAAAAuBAAAAAAAAAAACAAAAAAAl5IVp2O5R1smIeUIAAAAALU2DPUAAPQbAAAAAA==", expectedErr: ErrBadAttrVersion},
	} {
		_, err := ParseAttribution(tc.input)
		if err == nil {
			t.Errorf("expected error, input was: %s", tc.input)
		}

		if !errors.Is(err, tc.expectedErr) {
			t.Errorf("expected error: %s, got %s, input was: %s", tc.expectedErr, err, tc.input)
		}
	}
}
