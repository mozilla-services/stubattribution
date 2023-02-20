package dmgreader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"howett.net/plist"
)

// DMG is a structure representing a DMG file.
type DMG struct {
	name string
	file *os.File
}

// ReadAtSeeker is the interface that groups the basic ReadAt and Seek methods.
type ReadAtSeeker interface {
	io.ReaderAt
	io.Seeker
}

var (
	ErrNoPropertyList = errors.New("dmg: no XML property list")
)

// OpenFile returns a new instance to interact with a DMG file. This function
// might return an error if the file does not exist or if there is a problem
// opening it. When you create an instance of DMG, you must close it.
func OpenFile(name string) (*DMG, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("dmg: %w", err)
	}

	return &DMG{name: name, file: file}, nil
}

// Close closes the DMG file.
func (d *DMG) Close() error {
	if d.file != nil {
		return d.file.Close()
	}

	return nil
}

// ParseXMLPropertyList parses the XML property list pointed by the koly block.
func (d *DMG) ParseXMLPropertyList() (map[string]interface{}, error) {
	return parseXMLPropertyList(d.file)
}

func parseXMLPropertyList(input ReadAtSeeker) (map[string]interface{}, error) {
	var data map[string]interface{}

	// We need to know the offset/length of the XML property list in the DMG
	// file, which are conveniently stored in the "koly" block, assuming this
	// block is present.
	block, err := parseKolyBlock(input)
	if err != nil {
		return data, err
	}
	if block.XMLLength == 0 {
		return data, ErrNoPropertyList
	}

	buf := make([]byte, block.XMLLength)
	if _, err := input.ReadAt(buf, int64(block.XMLOffset)); err != nil {
		return data, fmt.Errorf("dmg: %w", err)
	}

	if err := plist.NewDecoder(bytes.NewReader(buf)).Decode(&data); err != nil {
		return data, fmt.Errorf("dmg: %w", err)
	}

	return data, nil
}
