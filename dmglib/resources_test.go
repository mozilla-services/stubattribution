package dmglib

import (
	"testing"
)

var TestResources = Resources{
	Entries: map[string][]ResourceData{
		"one": []ResourceData{ResourceData{
			Attributes: "attr1",
			CFName:     "",
			Data:       []uint8{1},
			ID:         "1",
			Name:       "one data",
		}},
		"two": []ResourceData{ResourceData{
			Attributes: "attr2",
			CFName:     "",
			Data:       []uint8{2},
			ID:         "2",
			Name:       "three data",
		}},
	},
}

func TestGetResourceByName(t *testing.T) {
	res, err := TestResources.GetResourceDataByName("one")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if len(res) != 1 {
		t.Errorf("wrong number of ResourceData entries: %d, expected 1", len(res))
	}

	if res[0].ID != "1" {
		t.Errorf("wrong ResourceData; got id: %s, expected 1", res[0].ID)
	}
}

func TestGetResourceByNameNotFound(t *testing.T) {
	_, err := TestResources.GetResourceDataByName("three")

	if err != ErrResourceNotFound {
		t.Errorf("expected ErrResourceNotFound, got: %s", err)
	}
}

var UnparsedResources = map[string]interface{}{
	// This blkx block isn't representative of a real one, but the values make it
	// easy to largely verify the parsing.
	"blkx": []map[string]interface{}{
		map[string]interface{}{
			"Attributes": "blkxattr",
			"CFName":     "blkxcfname",
			"Data":       []uint8{0, 1, 2, 3, 4},
			"ID":         "blkxid",
			"Name":       "blkxname",
		},
	},
	// This plst object is more representative of what we may see in the wild.
	// Most notably, attributable DMGs have a base64 encoded binary structure
	// buried in them.
	"plst": []map[string]interface{}{
		map[string]interface{}{
			"Attributes": "0x0050",
			"Data":       []uint8{0},
			"ID":         "0",
			"Name":       "\n\t\t\t\tAAAAAAAAAAA\n\t\t\t\tBBBBBBBBBBBBBB\n\t\t\t\t",
		},
	},
}

func TestParseResources(t *testing.T) {
	resources, err := parseResources(UnparsedResources)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	blkx := resources.Entries["blkx"][0]
	if blkx.Attributes != "blkxattr" {
		t.Errorf("unexpected value for blkx.Attributes, expected blkxattr, got %s", blkx.Attributes)
	}

	plst := resources.Entries["plst"][0]
	if plst.Name != "AAAAAAAAAAABBBBBBBBBBBBBB" {
		t.Errorf("unexpected value for plst.Name, expected AAAAAAAAAAABBBBBBBBBBBBBB, got %s", plst.Name)
	}
}
