package dmglib

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type UDIFChecksum struct {
	Type_   uint32
	Bitness uint32
	Data    [32]uint32
}

type BLKXRun struct {
	Type_       uint32
	Reserved    uint32
	SectorStart uint64
	SectorCount uint64
	CompOffset  uint64
	CompLength  uint64
}

type BLKXTable struct {
	FUDIFBlocksSignature      uint32
	InfoVersion               uint32
	FirstSectorNumber         uint64
	SectorCount               uint64
	DataStart                 uint64
	DecompressBufferRequested uint32
	BlocksDescriptor          uint32
	Reserved1                 uint32
	Reserved2                 uint32
	Reserved3                 uint32
	Reserved4                 uint32
	Reserved5                 uint32
	Reserved6                 uint32       // 64 bytes to here
	Checksum                  UDIFChecksum // 136 for this, so 200 total now
	BlocksRunCount            uint32       // 204 total now
}

// In an ideal world we'd keep the BLKXRuns in the BLKXTable, but because we read
// the data for each of these things separately, we cannot. (Go does not allow reading
// only part of a structure from binary data.) To avoid any confusion or mismatching, we
// keep this together in this container instead.
type BLKXContainer struct {
	Table *BLKXTable
	Runs  []BLKXRun
}

var (
	// The offset in the blkx data where the `Runs` metadata begins. This comes directly after the
	// BLKXTable metadata, so it is equal to the size of that data structure.
	// This is needed because the size of `Runs` depends on the value of `BlocksRunCount`,
	// so it must be read after we have sized an array to that value.
	blkxRunsOffset = 204
)

func ParseBlkxData(data []uint8) (*BLKXContainer, error) {
	container := new(BLKXContainer)
	table := new(BLKXTable)

	byteReader := bytes.NewReader(data)
	if err := binary.Read(byteReader, binary.BigEndian, table); err != nil {
		return container, fmt.Errorf("dmglib: %w", err)
	}

	container.Table = table

	runs := make([]BLKXRun, table.BlocksRunCount)
	byteReader.Seek(0, blkxRunsOffset)

	if err := binary.Read(byteReader, binary.BigEndian, runs); err != nil {
		return container, fmt.Errorf("dmglib: %w", err)
	}

	container.Runs = runs

	return container, nil
}
