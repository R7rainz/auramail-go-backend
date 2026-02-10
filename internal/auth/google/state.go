package google

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// StateManager issues and validates short-lived OAuth state tokens.
type StateManager struct {
	states map[string]time.Time
	mu     sync.RWMutex
}

func NewStateManager() *StateManager {
	return &StateManager{states: make(map[string]time.Time)}
}

// Generate returns a cryptographically random state string valid for 10 minutes.
func (sm *StateManager) Generate() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(buf)

	sm.mu.Lock()
	sm.states[state] = time.Now().Add(10 * time.Minute)
	sm.mu.Unlock()

	return state, nil
}

// Validate checks that the state exists and is not expired; it removes one-time tokens.
func (sm *StateManager) Validate(state string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	expiry, ok := sm.states[state]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(sm.states, state)
		return false
	}
	delete(sm.states, state)
	return true
}
