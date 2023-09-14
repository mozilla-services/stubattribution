package dmgmodify

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/mozilla-services/stubattribution/dmglib"
	"github.com/vimeo/go-util/crc32combine"
)

var (
	ErrSentinelMissing        = errors.New("dmgmodify: sentinel value not found")
	ErrBlkxResNotFound        = errors.New("dmgmodify: unable to find blkx resource to update")
	ErrCodeTooLong            = errors.New("dmgmodify: attribution code is too long")
	TAB                       = 0x9
	NUL                       = 0x0
	crcPolynomial      uint32 = 0xedb88320
)

// Update `dmg`, replacing the `sentinel` area with the provided `code`.
// This function is a port of the C implementation from libdmg-hfsplus
// (https://github.com/mozilla/libdmg-hfsplus/blob/a0a959bd25370c1c0a00c9ec525e3e78285adbf9/dmg/attribution.c#L209)
// Note: We explicitly do _not_ update the Attribution resource here, as it is
// not necessary, and makes for unnecessary work in the critical path of a
// new Firefox install. This does not impact the attribution of the build, but
// it does mean the build cannot be re-attributed later (which is not something
// that ever needs to happen).
func WriteAttributionCode(dmg *dmglib.DMG, sentinel string, code []byte) error {
	// First, pull the information we need to update the attribution code.
	// The blkx resource has some metadata that we need to update after
	// injecting the attribution code.
	blkxRes, err := dmg.Resources.GetResourceDataByName("blkx")
	if err != nil {
		return err
	}

	// The plst resource contains some metadata that help us quickly
	// locate the attribution data, and update the blkx and top level dmg
	// metadata.
	plstRes, err := dmg.Resources.GetResourceDataByName("plst")
	if err != nil {
		return err
	}

	attr, err := dmglib.ParseAttribution(plstRes[0].Name)
	if err != nil {
		return err
	}

	// Find the offset of the sentinel string within the raw block, if exists
	attrOffset := bytes.Index(dmg.Data[attr.RawPos:attr.RawPos+attr.RawLength], []byte(sentinel))
	if attrOffset == -1 {
		return ErrSentinelMissing
	}
	// Finally, calculate the overall offset for the attribution area within `dmg.Data`
	fullAttrOffset := int(attr.RawPos) + attrOffset

	// Zero out the attribution area
	codeOffset := fullAttrOffset + len(sentinel)
	paddingOffset := codeOffset
	for {
		if dmg.Data[paddingOffset] == byte(TAB) {
			dmg.Data[paddingOffset] = byte(NUL)
		} else {
			break
		}
		paddingOffset += 1
	}

	// Ensure the new code will fit in the attribution area
	if len(code) > paddingOffset-codeOffset {
		return ErrCodeTooLong
	}

	// Update the attribution area with the new attribution code
	copy(dmg.Data[codeOffset:codeOffset+len(code)], code[:])

	// Calculate the new CRC value for the entire raw block that the
	// attribution code is within.
	rawCrc := crc32.Checksum(
		dmg.Data[attr.RawPos:attr.RawPos+attr.RawLength],
		crc32.MakeTable(crcPolynomial),
	)
	// Calculate the new CRC values for the blkx checksum and Koly block checksum
	// This is done by combining 3 separate CRCs:
	// 1) The CRC of the data _prior_ to the block the attribution data is in.
	//    This CRC comes from the attribution metadata in the plst resource.
	// 2) The CRC of the block the attribution data is in, which we calculated
	//    just above.
	// 3) The CRC of the data _after_ the block the attribution data is in.
	//    This CRC also comes from the attribution metadata.
	newBlkxChecksum := crc32combine.CRC32Combine(
		crcPolynomial,
		crc32combine.CRC32Combine(crcPolynomial,
			attr.BeforeUncompressedChecksum,
			rawCrc,
			int64(attr.RawLength),
		),
		attr.AfterUncompressedChecksum,
		int64(attr.AfterUncompressedLength),
	)
	newDataChecksum := crc32combine.CRC32Combine(
		crcPolynomial,
		crc32combine.CRC32Combine(crcPolynomial,
			attr.BeforeCompressedChecksum,
			rawCrc,
			int64(attr.RawLength),
		),
		attr.AfterCompressedChecksum,
		int64(attr.AfterCompressedLength),
	)

	// At this point we've updated the raw attribution code in the dmg
	// but the metadata (resources and checksums) are invalid, and need
	// to be updated.

	// First, we need to find the correct blkx resource. (We assume the first
	// one with an HFS filesystem is the correct one.)
	blkxIndex := -1
	for i, res := range blkxRes {
		if strings.Contains(res.Name, "Apple_HFS") {
			blkxIndex = i
			break
		}
	}
	if blkxIndex == -1 {
		return ErrBlkxResNotFound
	}

	// Parse the blkx metadata into an updatable struct
	blkx, err := dmglib.ParseBlkxData(blkxRes[blkxIndex].Data)
	if err != nil {
		return fmt.Errorf("dmgmodify: %w", err)
	}

	// Update its checksum
	blkx.Table.Checksum.Data[0] = newBlkxChecksum

	// Update the serialized version of the `blkx` metadata in the `blkxRes`.
	buf := &bytes.Buffer{}
	err = binary.Write(buf, binary.BigEndian, blkx.Table)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.BigEndian, blkx.Runs)
	if err != nil {
		return err
	}
	blkxRes[blkxIndex].Data = buf.Bytes()

	// Update the DMG's parsed and raw data with the new blkx resource data.
	err = dmg.UpdateResource("blkx", blkxRes)
	if err != nil {
		return err
	}

	// Update the Koly block parsed and raw data as well
	err = dmg.UpdateKolyBlock(newDataChecksum)
	if err != nil {
		return fmt.Errorf("WriteAttributionCode: %w", err)
	}

	// At this point we've inserted the new attribution data and updated all
	// of the necessary metadata. Easy, right?!
	return nil
}
