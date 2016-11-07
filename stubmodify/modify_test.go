package stubmodify

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func buildMapped(totalLen int, peHeaderOffset uint32, peMagicNumber uint16, certTableOffset, certTableSize uint32, tag []byte) []byte {
	mapped := make([]byte, totalLen)
	optionalHeaderOffset := peHeaderOffset + 24
	certDirEntryOffset := int(optionalHeaderOffset) + 128
	if peMagicNumber == 0x20b {
		certDirEntryOffset = int(optionalHeaderOffset) + 144
	}

	binary.LittleEndian.PutUint32(mapped[0x3C:0x40], peHeaderOffset)
	binary.LittleEndian.PutUint16(mapped[optionalHeaderOffset:optionalHeaderOffset+2], peMagicNumber)

	if len(mapped) > certDirEntryOffset+8 {
		binary.LittleEndian.PutUint32(mapped[certDirEntryOffset:certDirEntryOffset+4], certTableOffset)
		binary.LittleEndian.PutUint32(mapped[certDirEntryOffset+4:certDirEntryOffset+8], certTableSize)
		copy(mapped[certTableOffset:int(certTableOffset)+len(tag)], tag)
	}

	return mapped
}

func TestWriteAttributionCodeBounds(t *testing.T) {
	// Shorter than peHeaderOffset
	t.Run("shorterThanPeHeader", func(t *testing.T) {
		_, err := WriteAttributionCode([]byte(""), []byte("a test code"))
		if err.Error() != fmt.Sprintf("mapped must be at least %d bytes", 0x40) {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	// Shorter than optionalHeaderOffset
	t.Run("shorterThanOptionalHeaderOffset", func(t *testing.T) {
		mapped := make([]byte, 0x40+64)
		binary.LittleEndian.PutUint32(mapped[0x3C:0x40], 100000)
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != fmt.Sprintf("mapped is shorter than optionalHeaderOffset+2: %d", 100000+26) {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	t.Run("badMagicNumber", func(t *testing.T) {
		mapped := buildMapped(0x80+100, 0x80, 0, 0x160, 0, []byte(""))
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != "mapped is not in a known PE format" {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	// Shorter than certDirEntryOffset
	t.Run("magicNumber0x10b", func(t *testing.T) {
		mapped := buildMapped(0x80+100, 0x80, 0x10b, 0x160, 0, []byte(""))
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != fmt.Sprintf("mapped is shorter than certDirEntryOffset+8: %d", 0x80+24+128+8) {
			t.Errorf("Incorrect error returned err: %v, expected %d bytes", err, 0x80+24+128+8)
		}
	})

	t.Run("magicNumber0x20b", func(t *testing.T) {
		mapped := buildMapped(0x80+100, 0x80, 0x20b, 0x160, 0, []byte(""))
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != fmt.Sprintf("mapped is shorter than certDirEntryOffset+8: %d", 0x80+24+144+8) {
			t.Errorf("Incorrect error returned err: %v, expected %d bytes", err, 0x80+24+144+8)
		}
	})

	t.Run("mappedNotSigned", func(t *testing.T) {
		mapped := buildMapped(0x80+1000, 0x80, 0x20b, 0x160, 0, []byte(""))
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != "mapped is not signed" {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	t.Run("shorterThanCertTableSize", func(t *testing.T) {
		mapped := buildMapped(0x80+1000, 0x80, 0x20b, 0x160, 1000, []byte(""))
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != fmt.Sprintf("mapped is shorter than certTableOffset+certTableSize: %d", 0x160+1000) {
			t.Errorf("Incorrect error returned err: %v, expected %d bytes", err, 0x80+24+144+8)
		}
	})

	t.Run("noDummyCert", func(t *testing.T) {
		mapped := buildMapped(0x80+1000, 0x80, 0x20b, 0x160, 300, []byte("FAIL"))
		_, err := WriteAttributionCode(mapped, []byte("a test code"))
		if err.Error() != "mapped does not contain dummy cert" {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	t.Run("tooMuchData", func(t *testing.T) {
		mapped := buildMapped(0x160+300, 0x80, 0x20b, 0x160, 300, []byte("__MOZCUSTOM__:"))
		_, err := WriteAttributionCode(mapped, make([]byte, 980))
		if err == nil || err.Error() != "we are trying to write past the end of mapped" {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	t.Run("writing outside of certtable", func(t *testing.T) {
		mapped := buildMapped(0x160+6000, 0x80, 0x20b, 0x160, 300, []byte("__MOZCUSTOM__:"))
		_, err := WriteAttributionCode(mapped, make([]byte, 980))
		if err == nil || err.Error() != "code is longer than available cert table space" {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	t.Run("code too long", func(t *testing.T) {
		_, err := WriteAttributionCode([]byte(""), make([]byte, 1025))
		if err.Error() != "code + __MOZCUSTOM__ exceeds 1024 bytes" {
			t.Errorf("Incorrect error returned err: %v", err)
		}
	})

	t.Run("succesful run", func(t *testing.T) {
		code := []byte("acustomcode")
		mappedLen := 0x160 + 6000
		mapped := buildMapped(mappedLen, 0x80, 0x20b, 0x160, 2000, []byte("__MOZCUSTOM__:"))
		modBytes, err := WriteAttributionCode(mapped, code)
		if err != nil {
			t.Errorf("Error returned: %s", err)
		}
		if len(modBytes) != mappedLen {
			t.Errorf("modBytes is the wrong size: %d", len(modBytes))
		}
		if !bytes.Contains(modBytes, code) {
			t.Errorf("modBytes does not contain code")
		}
	})
}
