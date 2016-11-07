package stubmodify

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// MozTag prefixes the attribution code
const MozTag = "__MOZCUSTOM__:"

// WriteAttributionCode inserts data into a prepared certificate in
// a signed PE file.
func WriteAttributionCode(mapped, code []byte) ([]byte, error) {
	if len(code)+len(MozTag) > 1024 {
		return nil, errors.New("code + __MOZCUSTOM__ exceeds 1024 bytes")
	}

	byteOrder := binary.LittleEndian

	// Get the location of the PE header and the option header
	if len(mapped) < 0x40 {
		return nil, fmt.Errorf("mapped must be at least %d bytes", 0x40)
	}
	peHeaderOffset := byteOrder.Uint32(mapped[0x3C:0x40])
	optionalHeaderOffset := peHeaderOffset + 24

	// Look up the magic number in the option header,
	// so we know if we have a 32 or 64-bit executable.
	// We need to know that so that we can find the data directories.
	if len(mapped) < int(optionalHeaderOffset+2) {
		return nil, fmt.Errorf("mapped is shorter than optionalHeaderOffset+2: %d", optionalHeaderOffset+2)
	}
	peMagicNumber := byteOrder.Uint16(mapped[optionalHeaderOffset : optionalHeaderOffset+2])

	var certDirEntryOffset uint32
	if peMagicNumber == 0x10b {
		certDirEntryOffset = optionalHeaderOffset + 128
	} else if peMagicNumber == 0x20b {
		certDirEntryOffset = optionalHeaderOffset + 144
	} else {
		return nil, errors.New("mapped is not in a known PE format")
	}

	if len(mapped) < int(certDirEntryOffset+8) {
		return nil, fmt.Errorf("mapped is shorter than certDirEntryOffset+8: %d", certDirEntryOffset+8)
	}
	certTableOffset := byteOrder.Uint32(mapped[certDirEntryOffset : certDirEntryOffset+4])
	certTableSize := byteOrder.Uint32(mapped[certDirEntryOffset+4 : certDirEntryOffset+8])

	if certTableOffset == 0 || certTableSize == 0 {
		return nil, errors.New("mapped is not signed")
	}

	tag := []byte(MozTag)
	if len(mapped) < int(certTableOffset+certTableSize) {
		return nil, fmt.Errorf("mapped is shorter than certTableOffset+certTableSize: %d", certTableOffset+certTableSize)
	}
	tagIndex := bytes.Index(mapped[certTableOffset:certTableOffset+certTableSize], tag)
	if tagIndex == -1 {
		return nil, errors.New("mapped does not contain dummy cert")
	}

	insertStart := int(certTableOffset) + tagIndex + len(tag)
	if insertStart+len(code) >= len(mapped) {
		return nil, errors.New("we are trying to write past the end of mapped")
	}

	if insertStart+len(code) > int(certTableOffset+certTableSize) {
		return nil, fmt.Errorf("code is longer than available cert table space")
	}

	modBytes := make([]byte, len(mapped))
	copy(modBytes, mapped)

	copy(modBytes[insertStart:insertStart+len(code)], code)

	return modBytes, nil
}
