package stubhandlers

import (
	"sync"
	"time"

	cache "github.com/mozilla-services/sizedlrucache"
)

type lockedCache struct {
	lru           *cache.SizedLRU
	cacheDuration time.Duration
	lck           sync.Mutex
}

func newLockedCache(maxSize int64, dur time.Duration) *lockedCache {
	return &lockedCache{
		lru:           cache.NewSizedLRU(maxSize),
		cacheDuration: dur,
	}
}

func (l *lockedCache) Add(key string, val interface{}, size int64) {
	l.lck.Lock()
	defer l.lck.Unlock()
	l.lru.Add(key, val, size, time.Now().Add(l.cacheDuration))
}

func (l *lockedCache) Get(key string) (interface{}, bool) {
	l.lck.Lock()
	defer l.lck.Unlock()
	return l.lru.Get(key)
}
