package engine

import (
	"net/http"
	"sync"
	"time"
)

type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
}

// Create a dedicated state struct for each idempotency key
type IdempotencyState struct {
	Resp *CachedResponse // Use a pointer so we can check if it is nil
	Mu   sync.RWMutex
}

// The Engine now holds pointers to the state, protected by a global map lock
type IdempotencyEngine struct {
	states   map[string]*IdempotencyState
	globalMu sync.RWMutex
}

func NewIdempotencyEngine() *IdempotencyEngine {
	return &IdempotencyEngine{
		states: make(map[string]*IdempotencyState),
	}
}

// Encapsulate the Double-Check Locking logic inside the engine
func (IE *IdempotencyEngine) GetState(key string) *IdempotencyState {
	// Fast path read
	IE.globalMu.RLock()
	state, exists := IE.states[key]
	IE.globalMu.RUnlock()

	if !exists {
		IE.globalMu.Lock()
		state, exists = IE.states[key]
		if !exists {
			state = &IdempotencyState{}
			IE.states[key] = state
		}
		IE.globalMu.Unlock()
	}

	return state
}

func (IE *IdempotencyEngine) CleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		threshold := time.Now().Add(-24 * time.Hour)

		IE.globalMu.Lock()
		for key, state := range IE.states {
			state.Mu.RLock()

			if state.Resp != nil && state.Resp.CreatedAt.Before(threshold) {
				state.Mu.RUnlock()
				delete(IE.states, key)
				continue
			}
			state.Mu.RUnlock()
		}
		IE.globalMu.Unlock()
	}
}
