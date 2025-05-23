package syncmap

import (
	"sync"
)

type SyncMap[K comparable, V any] struct {
	m sync.Map
}

func (s *SyncMap[K, V]) Store(key K, value V) {
	s.m.Store(key, value)
}

func (s *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	val, ok := s.m.Load(key)
	if ok {
		value = val.(V)
	}
	return
}

func (s *SyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	actualVal, loaded := s.m.LoadOrStore(key, value)
	actual = actualVal.(V)
	return
}

func (s *SyncMap[K, V]) Delete(key K) {
	s.m.Delete(key)
}

func (s *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	s.m.Range(func(k, v any) bool {
		return f(k.(K), v.(V))
	})
}
