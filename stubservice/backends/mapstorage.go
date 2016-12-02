package backends

import (
	"fmt"
	"io"
	"io/ioutil"
)

type MapStorageItem struct {
	ContentType string
	Bytes       []byte
}

// MapStorage is for testing purposes
type MapStorage struct {
	Storage map[string]MapStorageItem
}

func NewMapStorage() *MapStorage {
	return &MapStorage{
		Storage: make(map[string]MapStorageItem),
	}
}

func (m *MapStorage) Exists(key string) bool {
	_, ok := m.Storage[key]
	return ok
}

func (m *MapStorage) Put(key string, contentType string, body io.ReadSeeker) error {
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("error reading body: %s", err)
	}
	m.Storage[key] = MapStorageItem{contentType, bytes}
	return nil
}
