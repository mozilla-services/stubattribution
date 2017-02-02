package stubhandlers

import "time"

var globalStringCache = newStringCache(1024*1024*128, 5*time.Minute) // 128M

type stringCache struct {
	cache *lockedCache
}

func newStringCache(maxSize int64, dur time.Duration) *stringCache {
	return &stringCache{
		cache: newLockedCache(maxSize, dur),
	}
}

func (s *stringCache) Add(key string, val string) {
	s.cache.Add(key, val, int64(len(val)))
}

func (s *stringCache) Get(key string) string {
	if val, hit := s.cache.Get(key); hit {
		return val.(string)
	}
	return ""
}
