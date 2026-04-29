package ai_agent

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type ConfirmationStore interface {
	Save(action PendingAction) error
	Get(token string) (PendingAction, error)
	MarkConfirmed(token string) error
	Delete(token string) error
}

type InMemoryStore struct {
	mu      sync.RWMutex
	actions map[string]PendingAction
	ttl     time.Duration
}

func NewInMemoryStore(ttl time.Duration) *InMemoryStore {
	return &InMemoryStore{
		actions: make(map[string]PendingAction),
		ttl:     ttl,
	}
}

func (s *InMemoryStore) Save(action PendingAction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	action.CreatedAt = time.Now()
	s.actions[action.Token] = action

	go s.cleanup()
	return nil
}

func (s *InMemoryStore) Get(token string) (PendingAction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	action, exists := s.actions[token]
	if !exists {
		return PendingAction{}, ErrActionNotFound
	}
	return action, nil
}

func (s *InMemoryStore) MarkConfirmed(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	action, exists := s.actions[token]
	if !exists {
		return ErrActionNotFound
	}
	action.Confirmed = true
	s.actions[token] = action
	return nil
}

func (s *InMemoryStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.actions, token)
	return nil
}

func (s *InMemoryStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, action := range s.actions {
		if now.Sub(action.CreatedAt) > s.ttl {
			delete(s.actions, token)
		}
	}
}

func GenerateToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
