package dmglib

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
)

const (
	attrBlockSignature = "attr"
	attrBlockVersion   = 1
	attrBlockSize      = 76
)

var (
	ErrBadAttrBase64     = errors.New("attr: couldn't decode base64 data")
	ErrBadAttrBinaryData = errors.New("attr: couldn't parse binary attribution data")
	ErrBadAttrSignature  = errors.New("attr: invalid attribution signature")
)

type AttributionResource struct {
	Signature                  [4]byte
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

func parseAttribution(raw string) (*AttributionResource, error) {
	attr := new(AttributionResource)

	if raw == "" {
		return attr, nil
	}

	buf, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.ReplaceAll(raw, "\t", ""), "\n", ""))
	if err != nil {
		return attr, ErrBadAttrBase64
	}

	if err := binary.Read(bytes.NewReader(buf), binary.LittleEndian, attr); err != nil {
		return attr, ErrBadAttrBinaryData
	}

	if !bytes.Equal(attr.Signature[:], []byte(attrBlockSignature)) {
		return attr, ErrBadAttrSignature
	}

	return attr, nil
}
