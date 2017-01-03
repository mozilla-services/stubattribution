package stubhandlers

import (
	"sync"
	"time"

	"github.com/mozilla-services/stubattribution/cache"
)

type stub struct {
	contentType string
	body        []byte
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

func (s *stubCache) Add(key string, st *stub) {
	s.lck.Lock()
	defer s.lck.Unlock()

	size := int64(len(st.body) + len([]byte(st.contentType)))
	expires := time.Now().Add(s.cacheDuration)
	s.lru.Add(key, st, size, expires)
}

func (s *stubCache) Get(key string) *stub {
	s.lck.Lock()
	defer s.lck.Unlock()

	if val, hit := s.lru.Get(key); hit {
		return val.(*stub)
	}
	return nil
}
