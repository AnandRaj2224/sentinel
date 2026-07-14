package engine

import (
	"sync"
	"time"
)

type RouteConfig struct {
	TargetURLs     []string
	WindowSeconds int
	MaxRequests   int
}

// RateLimiter struct contains visitors
// that maps ip address to requests and
// the frequency / no of time in a time period.
type RateLimiter struct {
	visitors map[string][]time.Time
	mu       sync.RWMutex
}

// NewRateLimiter initalises a new RateLimiter struct
// and return it address.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string][]time.Time),
	}
}

// Allow implements a sliding window algo for
// blocking the requests that exceeds the limits.
func (rl *RateLimiter) Allow(ip string, windowSeconds int, maxRequests int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	var filtered []time.Time
	threshold := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	for _, t := range rl.visitors[ip] {
		if t.After(threshold) {
			filtered = append(filtered, t)
		}
	}
	rl.visitors[ip] = filtered
	if len(rl.visitors[ip]) < maxRequests {
		rl.visitors[ip] = append(rl.visitors[ip], time.Now())
		return true
	}
	return false
}

func (rl *RateLimiter) CleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)

	for range ticker.C {
		threshold := time.Now().Add(-10 * time.Second)
		rl.mu.Lock()
		for ip, timestamps := range rl.visitors {
			if len(timestamps) == 0 || timestamps[len(timestamps)-1].Before(threshold) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
