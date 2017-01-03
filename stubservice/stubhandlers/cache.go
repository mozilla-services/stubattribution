package stubhandlers

import (
	"sync"
	"time"

	"github.com/mozilla-services/stubattribution/cache"
)

var globalStubCache = newStubCache(
	1024*1024*1000, // 1G
	5*time.Minute,
)

type stub struct {
	contentType string
	body        []byte
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
	lru           *cache.SizedLRU
	cacheDuration time.Duration
	lck           sync.Mutex
}

func newStubCache(maxSize int64, dur time.Duration) *stubCache {
	return &stubCache{
		lru:           cache.NewSizedLRU(maxSize),
		cacheDuration: dur,
	}
}

// Add adds a new stub to the cache
func (s *stubCache) Add(key string, st *stub) {
	s.lck.Lock()
	defer s.lck.Unlock()

	size := int64(len(st.body) + len([]byte(st.contentType)))
	expires := time.Now().Add(s.cacheDuration)
	s.lru.Add(key, st, size, expires)
}

// Get returns a copy, if it exists so the original data is never corrupted
func (s *stubCache) Get(key string) *stub {
	s.lck.Lock()
	defer s.lck.Unlock()

	if val, hit := s.lru.Get(key); hit {
		return val.(*stub).copy()
	}
	return nil
}

// SetMaxSize sets a new maxsize, but does not resize the underlying cache
func (s *stubCache) SetMaxSize(maxSize int64) {
	s.lck.Lock()
	defer s.lck.Unlock()
	s.lru.MaxSize = maxSize
}
