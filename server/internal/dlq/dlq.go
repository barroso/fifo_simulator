package dlq

import (
	"fifo-simulator/server/internal/models"
	"sync"
)

// Store holds DLQ items in memory for the API.
type Store struct {
	mu    sync.RWMutex
	items []models.DLQItem
	cap   int
}

// NewStore creates a DLQ store with a max capacity (e.g. 500).
func NewStore(cap int) *Store {
	if cap <= 0 {
		cap = 500
	}
	return &Store{items: make([]models.DLQItem, 0, cap), cap: cap}
}

// Append adds an item (FIFO, drop oldest if at cap).
func (s *Store) Append(item models.DLQItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
	if len(s.items) > s.cap {
		s.items = s.items[len(s.items)-s.cap:]
	}
}

// All returns a copy of all items (newest last).
func (s *Store) All() []models.DLQItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.DLQItem, len(s.items))
	copy(out, s.items)
	return out
}

// Ensure Store implements kafka.DLQStore
var _ interface{ Append(models.DLQItem); All() []models.DLQItem } = (*Store)(nil)
