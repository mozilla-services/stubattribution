package stubattribution

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const MozTag = "__MOZCUSTOM__:"

// WriteAttributionCode inserts data into a prepared certificate in
// a signed PE file.
func WriteAttributionCode(mapped, code []byte) ([]byte, error) {
	if len(code)+len(MozTag) > 1024 {
		return nil, errors.New("code + __MOZCUSTOM__ exceeds 1024 bytes.")
	}

	modBytes := make([]byte, len(mapped))
	copy(modBytes, mapped)

	byteOrder := binary.LittleEndian

	// Get the location of the PE header and the option header
	peHeaderOffset := byteOrder.Uint32(modBytes[0x3C:0x40])
	optionalHeaderOffset := peHeaderOffset + 24

	// Look up the magic number in the option header,
	// so we know if we have a 32 or 64-bit executable.
	// We need to know that so that we can find the data directories.
	peMagicNumber := byteOrder.Uint16(modBytes[optionalHeaderOffset : optionalHeaderOffset+2])

	var certDirEntryOffset uint32
	if peMagicNumber == 0x10b {
		certDirEntryOffset = optionalHeaderOffset + 128
	} else if peMagicNumber == 0x20b {
		certDirEntryOffset = optionalHeaderOffset + 144
	} else {
		return nil, errors.New("mapped is not in a known PE format")
	}

	certTableOffset := byteOrder.Uint32(modBytes[certDirEntryOffset : certDirEntryOffset+4])
	certTableSize := byteOrder.Uint32(modBytes[certDirEntryOffset+4 : certDirEntryOffset+8])

	if certTableOffset == 0 || certTableSize == 0 {
		return nil, errors.New("mapped is not signed")
	}

	tag := []byte(MozTag)
	tagIndex := bytes.Index(modBytes[certTableOffset:certTableOffset+certTableSize], tag)
	if tagIndex == -1 {
		return nil, errors.New("mapped does not contain dummy cert")
	}

	insertStart := int(certTableOffset) + tagIndex + len(tag)
	if insertStart+len(code) >= len(modBytes) {
		return nil, errors.New("we are trying to write past the end of mapped")
	}
	copy(modBytes[insertStart:insertStart+len(code)], code)

	return modBytes, nil
}
