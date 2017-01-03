package cache

import (
	"container/list"
	"time"
)

type entry struct {
	key     string
	value   interface{}
	size    int64
	expires time.Time
}

type SizedLRU struct {
	cache   map[string]*list.Element
	ll      *list.List
	size    int64
	MaxSize int64
}

func NewSizedLRU(maxSize int64) *SizedLRU {
	return &SizedLRU{
		cache:   make(map[string]*list.Element),
		ll:      list.New(),
		size:    0,
		MaxSize: maxSize,
	}
}

// Get fetches item associated with key, otherwise returns nil
func (s *SizedLRU) Get(key string) (value interface{}, ok bool) {
	ele, hit := s.cache[key]
	if !hit {
		return
	}

	ent := ele.Value.(*entry)
	if time.Now().After(ent.expires) {
		s.removeElement(ele)
		return
	}

	s.ll.MoveToFront(ele)
	return ent.value, true
}

// Set adds an item to the cache
func (s *SizedLRU) Add(key string, val interface{}, size int64, expires time.Time) {
	if size > s.MaxSize {
		// val is too big for this cache
		return
	}
	defer s.prune()

	if e, ok := s.cache[key]; ok {
		s.ll.MoveToFront(e)

		ent := e.Value.(*entry)
		s.size += size - ent.size
		ent.size = size
		ent.value = val
		ent.expires = expires

		return
	}

	ele := s.ll.PushFront(&entry{key, val, size, expires})
	s.cache[key] = ele
	s.size += size

}

func (s *SizedLRU) prune() {
	for s.size > s.MaxSize {
		s.removeOldest()
	}
}

func (s *SizedLRU) removeOldest() {
	ele := s.ll.Back()
	if ele != nil {
		s.removeElement(ele)
	}
}

func (s *SizedLRU) removeElement(ele *list.Element) {
	s.ll.Remove(ele)
	ent := ele.Value.(*entry)
	delete(s.cache, ent.key)
	s.size -= ent.size
}
