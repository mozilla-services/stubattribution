package stubhandlers

import "time"

var globalStubCache = newStubCache(
	1024*1024*1000, // 1G
	5*time.Minute,
)

type stub struct {
	body        []byte
	contentType string
	filename    string
}

func (s *stub) copy() *stub {
	b := make([]byte, len(s.body))
	copy(b, s.body)
	return &stub{
		contentType: s.contentType,
		body:        b,
	}
}

type stubCache struct {
	cache *lockedCache
}

func newStubCache(maxSize int64, dur time.Duration) *stubCache {
	return &stubCache{
		cache: newLockedCache(maxSize, dur),
	}
}

// Add adds a new stub to the cache
func (s *stubCache) Add(key string, st *stub) {
	size := int64(len(st.body) + len([]byte(st.contentType)))
	s.cache.Add(key, st, size)
}

// Get returns a copy, if it exists so the original data is never corrupted
func (s *stubCache) Get(key string) *stub {
	if val, hit := s.cache.Get(key); hit {
		return val.(*stub).copy()
	}
	return nil
}
