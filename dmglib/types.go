package dmglib

import (
	"io"
)

// ReaderSeeker groups the basic Reader, ReaderAt and Seeker methods
type ReaderSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

// ReaderAtSeeker groups the basic ReaderAt and Seeker methods
type ReaderAtSeeker interface {
	io.ReaderAt
	io.Seeker
}
