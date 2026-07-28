package engine

import (
	"sync"
	"time"
)

type RouteConfig struct {
	TargetURLs    []string
	WindowSeconds int
	MaxRequests   int
}

// visitorState bundles the timestamps and a dedicated lock for a single IP.
type visitorState struct {
	timestamps []time.Time
	mu         sync.Mutex
}

// RateLimiter tracks client request timestamps in a thread-safe map.
type RateLimiter struct {
	visitors map[string]*visitorState
	globalMu sync.RWMutex
}

// NewRateLimiter initializes a new RateLimiter struct.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitorState),
	}
}

// Allow implements a sliding window algorithm.
func (rl *RateLimiter) Allow(ip string, windowSeconds int, maxRequests int) bool {
	rl.globalMu.RLock()
	state, exists := rl.visitors[ip]
	rl.globalMu.RUnlock()

	if !exists {
		rl.globalMu.Lock()
		state, exists = rl.visitors[ip]
		if !exists {
			state = &visitorState{
				timestamps: make([]time.Time, 0),
			}
			rl.visitors[ip] = state
		}
		rl.globalMu.Unlock()
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	var filtered []time.Time
	threshold := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	for _, t := range state.timestamps {
		if t.After(threshold) {
			filtered = append(filtered, t)
		}
	}

	state.timestamps = filtered

	if len(state.timestamps) < maxRequests {
		state.timestamps = append(state.timestamps, time.Now())
		return true
	}

	return false
}

// CleanupWorker removes stale IPs from the map.
func (rl *RateLimiter) CleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		threshold := time.Now().Add(-10 * time.Second)

		// Lock the global map to safely delete old entries
		rl.globalMu.Lock()
		for ip, state := range rl.visitors {
			state.mu.Lock()
			if len(state.timestamps) == 0 || state.timestamps[len(state.timestamps)-1].Before(threshold) {
				delete(rl.visitors, ip)
			}
			state.mu.Unlock()
		}
		rl.globalMu.Unlock()
	}
}
