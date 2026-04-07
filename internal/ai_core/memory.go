package ai_core

import (
	"sync"
	"time"
)

// MemoryManager abstracts conversation storage across sessions.
type MemoryManager interface {
	GetHistory(sessionID string) []ReactObservation
	AddObservation(sessionID string, obs ReactObservation)
	Clear(sessionID string)
}

// transientMemory implements MemoryManager using an in-memory map with TTL expiration.
// Ideal for V1 implementations avoiding complex DB schemas.
type transientMemory struct {
	mu       sync.RWMutex
	sessions map[string]sessionStore
	ttl      time.Duration
}

type sessionStore struct {
	history    []ReactObservation
	lastAccess time.Time
}

func NewTransientMemory(ttl time.Duration) MemoryManager {
	m := &transientMemory{
		sessions: make(map[string]sessionStore),
		ttl:      ttl,
	}
	go m.cleanupLoop()
	return m
}

func (m *transientMemory) GetHistory(sessionID string) []ReactObservation {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, exists := m.sessions[sessionID]
	if !exists {
		return nil
	}
	
	// Copy slice to prevent race conditions during iteration
	history := make([]ReactObservation, len(store.history))
	copy(history, store.history)
	
	store.lastAccess = time.Now()
	m.sessions[sessionID] = store
	return history
}

func (m *transientMemory) AddObservation(sessionID string, obs ReactObservation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := m.sessions[sessionID]
	store.history = append(store.history, obs)
	
	// Keep history bounded (e.g. max 50 observations per session to prevent context bounds errors)
	if len(store.history) > 50 {
		store.history = store.history[10:] // prune oldest 10 to keep context alive
	}
	
	store.lastAccess = time.Now()
	m.sessions[sessionID] = store
}

func (m *transientMemory) Clear(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// cleanupLoop periodically sweeps stale memory to prevent leaks.
func (m *transientMemory) cleanupLoop() {
	ticker := time.NewTicker(m.ttl / 2)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, store := range m.sessions {
			if now.Sub(store.lastAccess) > m.ttl {
				delete(m.sessions, id)
			}
		}
		m.mu.Unlock()
	}
}
