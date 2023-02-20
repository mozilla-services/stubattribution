package dmgreader

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// KolyBlock is a structure representing the "koly" block in a DMG file.
//
// See: http://newosxbook.com/DMG.html
type KolyBlock struct {
	Signature             [4]byte
	Version               uint32
	HeaderSize            uint32
	Flags                 uint32
	RunningDataForkOffset uint64
	DataForkOffset        uint64
	DataForkLength        uint64
	RsrcForkOffset        uint64
	RsrcForkLength        uint64
	SegmentNumber         uint32
	SegmentCount          uint32
	SegmentID             [4]uint32
	DataChecksumType      uint32
	DataChecksumSize      uint32
	DataChecksum          [32]uint32
	// XMLOffset is the offset of the property list in the DMG (from beginning).
	XMLOffset uint64
	// XMLLength is the length of the property list.
	XMLLength    uint64
	Reserved1    [120]uint8
	ChecksumType uint32
	ChecksumSize uint32
	Checksum     [32]uint32
	ImageVariant uint32
	SectorCount  uint64
	Reserved2    uint32
	Reserved3    uint32
	Reserved4    uint32
}

const (
	kolyBlockMagic = "koly"
	kolyBlockSize  = 512
)

var (
	ErrInvalidHeaderSize = errors.New("dmg: invalid header size")
	ErrNotKolyBlock      = errors.New("dmg: not a koly block")
)

func newKolyBlock() KolyBlock {
	block := KolyBlock{HeaderSize: kolyBlockSize}
	copy(block.Signature[:], kolyBlockMagic)

	return block
}

func parseKolyBlock(input ReadAtSeeker) (*KolyBlock, error) {
	block := new(KolyBlock)

	// Get the offset from the end of the DMG file minus 512 bytes, which is
	// where the koly block should be.
	offset, err := input.Seek(int64(-kolyBlockSize), io.SeekEnd)
	if err != nil {
		return block, fmt.Errorf("dmg: %w", err)
	}

	buf := make([]byte, kolyBlockSize)
	if _, err := input.ReadAt(buf, offset); err != nil {
		return block, fmt.Errorf("dmg: %w", err)
	}

	// From http://newosxbook.com/DMG.html, all fields in the koly block are in
	// big endian. This is to preserve compatibility with older generations of
	// PPC-based OS X.
	if err := binary.Read(bytes.NewReader(buf), binary.BigEndian, block); err != nil {
		return block, fmt.Errorf("dmg: %w", err)
	}

	if !bytes.Equal(block.Signature[:], []byte(kolyBlockMagic)) {
		return block, ErrNotKolyBlock
	}

	if int(block.HeaderSize) != kolyBlockSize {
		return block, ErrInvalidHeaderSize
	}

	return block, nil
}

func (b KolyBlock) write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, b)
}
