package maps

import (
	"sync"

	"github.com/google/uuid"
)

type RWMap struct {
	m   map[uuid.UUID]int
	mux sync.Mutex
}

func NewRWMap() *RWMap {
	return &RWMap{
		m: make(map[uuid.UUID]int),
	}
} // Set safely sets a value in the map

func (s *RWMap) Set(key uuid.UUID, pId int) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.m[key] = pId
}

// Get safely gets a value from the map
func (s *RWMap) Get(key uuid.UUID) (int, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	val, ok := s.m[key]
	return val, ok
}

// Delete safely removes a value from the map
func (s *RWMap) Delete(key uuid.UUID) {
	s.mux.Lock()
	defer s.mux.Unlock()
	delete(s.m, key)
}
