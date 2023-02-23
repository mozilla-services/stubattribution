package stubmodify

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

var maxQuickByteLen = 1024 * 64

type ByteGenerator []byte

func (b ByteGenerator) Generate(rand *rand.Rand, size int) reflect.Value {
	ran := make([]byte, rand.Intn(maxQuickByteLen))
	for i := 0; i < len(ran); i++ {
		ran[i] = byte(rand.Intn(256))
	}
	return reflect.ValueOf(ran)
}

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

func BenchmarkWriteAttributionCodeFull(b *testing.B) {
	fileBytes, err := ioutil.ReadFile("../testdata/test-stub.exe")
	if err != nil {
		b.Fatal("reading test-stub.exe", err)
	}
	code := []byte("testattributioncode&stuff")
	for i := 0; i < b.N; i++ {
		_, err := WriteAttributionCode(fileBytes, code)
		if err != nil {
			b.Error(err)
		}
	}
}

// TestWriteAttributionCodeFull tests WriteAttributionCode with a real binary
func TestWriteAttributionCodeFull(t *testing.T) {
	// test-stub.exe is a _slightly_ less than realistic file in that
	// its entire attribution area has been filled with non-nul data that
	// is not exactly what we would see in a real installer. filling up
	// the entire attribution area makes it easy for us to verify that any
	// existing attribution data will be properly removed before new data is added.
	fileBytes, err := ioutil.ReadFile("../testdata/test-stub.exe")
	if err != nil {
		t.Fatal("reading test-stub.exe", err)
	}

	origBytes := make([]byte, len(fileBytes))
	copy(origBytes, fileBytes)

	testWriteCode := func(t *testing.T, code []byte) {
		modBytes, err := WriteAttributionCode(fileBytes, code)
		if err != nil {
			t.Fatal("writing attribution code", err)
		}

		// Check that the origin byte slice was not modified
		if !bytes.Equal(origBytes, fileBytes) {
			t.Error("fileBytes was modified in WriteAttributionCode")
		}

		// Ensure the size didn't change.
		if len(modBytes) != len(fileBytes) {
			t.Errorf("modBytes: %d != fileBytes: %d", modBytes, fileBytes)
		}

		// Ensure the attribution code is included in the modified bytes.
		if !bytes.Contains(modBytes, code) {
			t.Errorf("modBytes does not contain code: %v", code)
		}

		// This is not as robust as the way WriteAttributionCode finds the offset
		// but it is likely good enough for tests -- and at the very least, means
		// that we will not have false negatives or positives if that implementation
		// is broken. Note that we use LastIndex because `MozTag` may be present
		// multiple times in a file, but the last instance is where attribution
		// data is written to.
		tagAndCode := make([]byte, 0)
		tagAndCode = append(tagAndCode, []byte(MozTag)...)
		tagAndCode = append(tagAndCode, code...)
		attrOffset := bytes.LastIndex(modBytes, tagAndCode)
		// Ensure everything after the tag and code ended up zeroed out.
		nulStart := attrOffset + len(MozTag) + len(code)
		nulEnd := attrOffset + MaxLength
		unusedAttrSpace := modBytes[nulStart:nulEnd]
		nulsFound := bytes.Count(unusedAttrSpace, []byte("\x00"))
		if nulsFound != nulEnd-nulStart {
			t.Errorf("Expected %v nuls, found %v", nulEnd-nulStart, nulsFound)
			t.Errorf("Instead, found this data at offset %v: %v", attrOffset, unusedAttrSpace)
		}

		// Check the next 10 bytes after the attribution space to ensure it was
		// unmodified.
		posAfterAttribution := nulEnd

		for i := 0; i < 10; i++ {
			pos := posAfterAttribution + i
			if fileBytes[pos] != modBytes[pos] {
				t.Errorf("At position %d, expected: 0x%02x but got: 0x%02x", pos, fileBytes[pos], modBytes[pos])
			}
		}
	}

	t.Run("static code test", func(t *testing.T) {
		testWriteCode(t, []byte("a test code"))
	})

	t.Run("fuzz code test", func(t *testing.T) {
		f := func(code []byte) bool {
			testWriteCode(t, code)
			return true
		}

		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
}

func TestWriteAttributionCodeBounds(t *testing.T) {
	t.Run("fuzz for panics", func(t *testing.T) {
		f := func(mapped, code ByteGenerator) bool {
			WriteAttributionCode(mapped, code)
			return true
		}
		if err := quick.Check(f, nil); err != nil {
			t.Error(err)
		}
	})
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
