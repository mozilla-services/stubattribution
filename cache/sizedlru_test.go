package cache

import (
	"fmt"
	"testing"
)

func TestSizedLRU(t *testing.T) {

	t.Run("max size key", func(t *testing.T) {
		cache := NewSizedLRU(64)
		cache.Add("testkey", "astring", 64)
		if len(cache.cache) != 1 {
			t.Errorf("Cache len: %d, expected 1", len(cache.cache))
		}
		if cache.size != 64 {
			t.Errorf("Cache size: %d, expected 64", cache.size)
		}

		cache.Add("testkey2", "astring", 1)
		if len(cache.cache) != 1 {
			t.Errorf("Cache len: %d, expected 1", len(cache.cache))
		}
		if cache.size != 1 {
			t.Errorf("Cache size: %d, expected 1", cache.size)
		}

		cache = NewSizedLRU(64)
		cache.Add("bigkey", "", 65)
		if len(cache.cache) != 0 {
			t.Errorf("Cache len: %d, expected 1", len(cache.cache))
		}
	})

	t.Run("multiple keys", func(t *testing.T) {
		cache := NewSizedLRU(64)
		for i := 1; i <= 32; i++ {
			cache.Add(fmt.Sprintf("testkey:%d", i), "astring", 2)
			if len(cache.cache) != i {
				t.Errorf("Cache len: %d, expected %d", len(cache.cache), i)
			}
			if cache.size != int64(i*2) {
				t.Errorf("Cache size: %d, expected %d", cache.size, i*2)
			}
		}

		if _, ok := cache.Get("testkey:1"); !ok {
			t.Errorf("testkey:1 should return ok")
		}

		cache.Add("testkey:33", "astring", 2)
		if len(cache.cache) != 32 {
			t.Errorf("Cache len: %d, expected %d", len(cache.cache), 32)
		}
		if cache.size != 64 {
			t.Errorf("Cache size: %d, expected %d", cache.size, 64)
		}

		if _, ok := cache.Get("testkey:2"); ok {
			t.Errorf("testkey:2 should not return ok")
		}

		cache = NewSizedLRU(64)
		for i := 1; i <= 32; i++ {
			cache.Add("testkey", "astring", 2)
			if val, _ := cache.Get("testkey"); val.(string) != "astring" {
				t.Errorf(`testkey was "%v", expected "astring"`, val)
			}
			if len(cache.cache) != 1 {
				t.Errorf("Cache len: %d, expected %d", len(cache.cache), 1)
			}
			if cache.size != 2 {
				t.Errorf("Cache size: %d, expected %d", cache.size, 2)
			}
		}
	})

}
