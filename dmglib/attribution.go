package dmglib

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	attrBlockSignature = 0x61747472 // "attr"
	attrBlockVersion   = 1
	attrBlockSize      = 76
)

var (
	ErrBadAttrBase64     = errors.New("dmglib: couldn't decode base64 data")
	ErrBadAttrLength     = errors.New("dmglib: bad object length")
	ErrBadAttrBinaryData = errors.New("dmglib: couldn't parse binary attribution data")
	ErrBadAttrSignature  = errors.New("dmglib: invalid attribution signature")
	ErrBadAttrVersion    = errors.New("dmglib: invalid attribution resource version")
)

type AttributionResource struct {
	Signature                  uint32
	Version                    uint32
	BeforeCompressedChecksum   uint32
	BeforeCompressedLength     uint64
	BeforeUncompressedChecksum uint32
	BeforeUncompressedLength   uint64
	RawPos                     uint64
	RawLength                  uint64
	RawChecksum                uint32
	AfterCompressedChecksum    uint32
	AfterCompressedLength      uint64
	AfterUncompressedChecksum  uint32
	AfterUncompressedLength    uint64
}

func ParseAttribution(raw string) (*AttributionResource, error) {
	attr := new(AttributionResource)

	if raw == "" {
		return attr, nil
	}

	// Raw attribution metadata strings sometime have tabs or newlines in them due to
	// being stored as a string in the plist. Get rid of them before decoding the data.
	buf, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.ReplaceAll((raw), "\t", ""), "\n", ""))
	if err != nil {
		return attr, ErrBadAttrBase64
	}

	if len(buf) != attrBlockSize {
		return attr, ErrBadAttrLength
	}

	// Read the data into a useful structure
	if err := binary.Read(bytes.NewReader(buf), binary.LittleEndian, attr); err != nil {
		return attr, fmt.Errorf("dmglib: %w", err)
	}

	// Sanity checks
	if attr.Signature != attrBlockSignature {
		return attr, ErrBadAttrSignature
	}

	if attr.Version != attrBlockVersion {
		return attr, ErrBadAttrVersion
	}

	return attr, nil
}
