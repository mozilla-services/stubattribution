package dmglib

import (
	"strings"
)

func makeInput(headerSize uint32) string {
	var sb strings.Builder

	block := newKolyBlock()
	block.HeaderSize = headerSize
	block.write(&sb)

	return sb.String()
}

func makeValidInput() string {
	return makeInput(kolyBlockSize)
}

func makeInvalidInputWithPropertyList() string {
	var sb strings.Builder

	// Add some padding at the beginning.
	for i := 0; i < 10; i++ {
		sb.WriteByte(0)
	}

	block := newKolyBlock()
	block.Version = 4
	block.XMLOffset = 2
	block.XMLLength = 1000 // This is clearly invalid.
	block.write(&sb)

	return sb.String()
}
