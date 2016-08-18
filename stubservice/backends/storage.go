package backends

import "io"

// Storage is an interface for storing objects
type Storage interface {
	Exists(key string) bool
	Put(key string, contentType string, body io.ReadSeeker) error
}
