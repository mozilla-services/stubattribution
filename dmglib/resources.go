package dmglib

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type ResourceData struct {
	Attributes string
	CFName     string
	Data       []uint8
	ID         string
	Name       string
}

type Resources struct {
	Entries map[string][]ResourceData
}

var (
	ErrResourceNotFound = errors.New("dmglib: named resource not found")
)

func parseResources(unparsed map[string]interface{}) (*Resources, error) {
	res := new(Resources)
	err := mapstructure.Decode(unparsed, &res.Entries)
	if err != nil {
		return res, fmt.Errorf("dmglib: %w", err)
	}

	return res, nil
}

func (r *Resources) GetResourceDataByName(name string) ([]ResourceData, error) {
	for k, v := range r.Entries {
		if k == name {
			return v, nil
		}
	}

	return []ResourceData{}, ErrResourceNotFound
}

func (r *Resources) UpdateByName(name string, data []ResourceData) {
	r.Entries[name] = data
}
