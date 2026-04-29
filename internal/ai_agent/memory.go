package ai_agent

import (
	"sync"
	"time"
)

type TransientMemory struct {
	mu      sync.RWMutex
	history map[string][]ReactObservation
	ttl     time.Duration
	access  map[string]time.Time
}

func NewTransientMemory(ttl time.Duration) *TransientMemory {
	return &TransientMemory{
		history: make(map[string][]ReactObservation),
		access:  make(map[string]time.Time),
		ttl:     ttl,
	}
}

func (m *TransientMemory) AddObservation(sessionID string, obs ReactObservation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionID == "" {
		return
	}

	m.history[sessionID] = append(m.history[sessionID], obs)
	m.access[sessionID] = time.Now()

	go m.cleanup()
}

func (m *TransientMemory) GetHistory(sessionID string) []ReactObservation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sessionID == "" {
		return nil
	}

	h, ok := m.history[sessionID]
	if !ok {
		return nil
	}
	
	cpy := make([]ReactObservation, len(h))
	copy(cpy, h)
	return cpy
}

func (m *TransientMemory) ClearHistory(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.history, sessionID)
	delete(m.access, sessionID)
}

func (m *TransientMemory) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for sessionID, lastAccess := range m.access {
		if now.Sub(lastAccess) > m.ttl {
			delete(m.history, sessionID)
			delete(m.access, sessionID)
		}
	}
}
