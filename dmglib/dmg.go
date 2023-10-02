package dmglib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	"github.com/mitchellh/mapstructure"
	"howett.net/plist"
)

var (
	ErrNoPropertyList         = errors.New("dmglib: no XML property list")
	ErrNoResourceFork         = errors.New("dmglib: no resource fork")
	ErrResourcesTooBig        = errors.New("dmglib: encoded resources are too big to be written")
	blkxUDIFCRC32      uint32 = 0x00000002
	blkxUDIFCRC32Size  uint32 = 32
)

// DMG is a structure representing a DMG file and its related metadata.
type DMG struct {
	Koly      *KolyBlock
	Resources *Resources
	Data      []byte
}

func (d *DMG) UpdateResource(name string, data []ResourceData) error {
	d.Resources.UpdateByName(name, data)
	return d.WriteResources()
}

func (d *DMG) UpdateKolyBlock(newDataChecksum uint32) error {
	// Update the data checksum
	d.Koly.DataChecksumType = blkxUDIFCRC32
	d.Koly.DataChecksumSize = blkxUDIFCRC32Size
	d.Koly.DataChecksum[0] = newDataChecksum

	// And update the overall checksum.
	err := d.UpdateOverallChecksum()
	if err != nil {
		return fmt.Errorf("UpdateKolyBlock: %w", err)
	}

	// Finally, write the changes we made to `d.Koly` to `d.Data`.
	d.WriteKolyBlock()

	return nil
}

// Ported from libdmg-hfsplus
// (https://github.com/mozilla/libdmg-hfsplus/blob/a0a959bd25370c1c0a00c9ec525e3e78285adbf9/dmg/dmglib.c#L50)
func (d *DMG) UpdateOverallChecksum() error {
	blkx, err := d.Resources.GetResourceDataByName("blkx")
	if err != nil {
		return fmt.Errorf("CalculateOverallChecksum: %w", err)
	}

	blkxData := make([]*BLKXContainer, len(blkx))
	for i, b := range blkx {
		blkxData[i], err = ParseBlkxData(b.Data)
		if err != nil {
			return fmt.Errorf("CalculateOverallChecksum: %w", err)
		}
		if blkxData[i].Table.Checksum.Type_ == blkxUDIFCRC32 {
			i += 1
		}
	}

	buf := make([]byte, len(blkxData)*4)
	for i := range blkxData {
		if blkxData[i].Table.Checksum.Type_ == blkxUDIFCRC32 {
			buf[(i*4)+0] = byte((blkxData[i].Table.Checksum.Data[0] >> 24) & 0xff)
			buf[(i*4)+1] = byte((blkxData[i].Table.Checksum.Data[0] >> 16) & 0xff)
			buf[(i*4)+2] = byte((blkxData[i].Table.Checksum.Data[0] >> 8) & 0xff)
			buf[(i*4)+3] = byte((blkxData[i].Table.Checksum.Data[0] >> 0) & 0xff)
		}
	}

	d.Koly.ChecksumType = blkxUDIFCRC32
	d.Koly.ChecksumSize = blkxUDIFCRC32Size
	d.Koly.Checksum[0] = crc32.Checksum(buf, crc32.MakeTable(0xedb88320))

	return nil
}

// Update the encoded resources in the raw data block with whatever
// is present in d.Resources.
func (d *DMG) WriteResources() error {
	var resourceMap map[string]interface{}
	err := mapstructure.Decode(d.Resources.Entries, &resourceMap)
	if err != nil {
		return err
	}
	var resourceData = make(map[string]interface{})
	resourceData["resource-fork"] = resourceMap

	buf := &bytes.Buffer{}
	enc := plist.NewEncoder(buf)
	enc.Indent("\n")
	err = enc.Encode(resourceData)
	if err != nil {
		return err
	}
	xml_len := int(d.Koly.XMLLength)
	// This shouldn't be possible - but if the new encoded resources are larger
	// than the original XML length, we cannot safely update them.
	if buf.Len() > xml_len {
		return ErrResourcesTooBig
	}
	// Pad the new resources with extra spaces to ensure they are exactly the
	// same length as the original ones. Failure to do so may cause some of
	// the original resources to stick around, and break the new resources.
	if buf.Len() < xml_len {
		need := xml_len - buf.Len()
		padding := make([]byte, need)
		padding[0] = 0x20
		for n := 1; n < need; n *= 2 {
			copy(padding[n:], padding[:n])
		}
		buf.Write(padding)
	}

	// Update the resources in the raw data block.
	copy(d.Data[d.Koly.XMLOffset:d.Koly.XMLOffset+d.Koly.XMLLength], buf.Bytes())

	return nil
}

// Update the Koly block in d.Data with whatever is present in d.Koly
func (d *DMG) WriteKolyBlock() error {
	buf := &bytes.Buffer{}

	if err := binary.Write(buf, binary.BigEndian, d.Koly); err != nil {
		return fmt.Errorf("WriteKolyBlock: %w", err)
	}

	if buf.Len() != kolyBlockSize {
		return fmt.Errorf("bad koly block size")
	}

	copy(d.Data[len(d.Data)-kolyBlockSize:], buf.Bytes())

	return nil
}

// DMGFile is a structure representing a DMG file on a filesystem.
type DMGFile struct {
	name string
	file *os.File
}

// OpenFile returns a new instance to interact with a DMG file. This function
// might return an error if the file does not exist or if there is a problem
// opening it. When you create an instance of DMG, you must close it.
func OpenFile(name string) (*DMGFile, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("dmglib: %w", err)
	}

	return &DMGFile{name: name, file: file}, nil
}

func (d *DMGFile) Parse() (*DMG, error) {
	return ParseDMG(d.file)
}

// Close closes the DMG file.
func (d *DMGFile) Close() error {
	if d.file != nil {
		return d.file.Close()
	}

	return nil
}

func ParseDMG(input ReaderSeeker) (*DMG, error) {
	dmg := new(DMG)

	// Parse the Koly block, which contains information we need to parse
	// the rest of the file.
	block, err := parseKolyBlock(input)
	if err != nil {
		return dmg, fmt.Errorf("dmglib: %w", err)
	}

	if block.XMLLength == 0 {
		return dmg, ErrNoPropertyList
	}

	// Read in the XML plist data
	buf := make([]byte, block.XMLLength)
	_, err = input.ReadAt(buf, int64(block.XMLOffset))
	if err != nil {
		return dmg, fmt.Errorf("dmglib: %w", err)
	}

	// Parse the XML plist into something structured
	var data map[string]interface{}
	if err := plist.NewDecoder(bytes.NewReader(buf)).Decode(&data); err != nil {
		return dmg, fmt.Errorf("dmglib: %w", err)
	}

	fork, ok := data["resource-fork"].(map[string]interface{})

	if !ok {
		return dmg, ErrNoResourceFork
	}

	// Transform the structured plist data into a proper structure
	resources, err := parseResources(fork)
	if err != nil {
		return dmg, fmt.Errorf("dmglib: %w", err)
	}

	// Read _all_ of the raw DMG data (this includes the raw bytes of things
	// such as the Koly block that we've already parsed).
	input.Seek(0, io.SeekStart)
	dmgData, err := io.ReadAll(input)
	if err != nil {
		return dmg, fmt.Errorf("dmglib: %w", err)
	}

	dmg.Koly = block
	dmg.Resources = resources
	dmg.Data = dmgData

	return dmg, nil
}
