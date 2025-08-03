package store

import (
	"sync"
	"time"
	"telemetry-demo/models"
)

type MemoryStore struct {
	subscribers map[int]*models.Subscriber
	nextID      int
	mu          sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		subscribers: make(map[int]*models.Subscriber),
		nextID:      1,
	}
}

func (s *MemoryStore) CreateSubscriber(name, email string) *models.Subscriber {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	subscriber := &models.Subscriber{
		ID:      s.nextID,
		Name:    name,
		Email:   email,
		Created: time.Now(),
	}
	
	s.subscribers[s.nextID] = subscriber
	s.nextID++
	
	return subscriber
}

func (s *MemoryStore) GetSubscriber(id int) (*models.Subscriber, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	subscriber, exists := s.subscribers[id]
	return subscriber, exists
}

func (s *MemoryStore) GetAllSubscribers() []*models.Subscriber {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	subscribers := make([]*models.Subscriber, 0, len(s.subscribers))
	for _, subscriber := range s.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	
	return subscribers
}