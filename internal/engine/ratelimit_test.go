package engine

import (
	"sync"
	"testing"
)

func TestRateLimiter_Sequential(t *testing.T) {
	limiter := NewRateLimiter()
	address := "1.1.1.1"
	reqWindow := 10
	limit := 2

	for i := range limit {
		if !limiter.Allow(address, reqWindow, limit) {
			t.Errorf("Request %d: should not have been blocked", i+1)
		}
	}
	if limiter.Allow(address, reqWindow, limit) {
		t.Errorf("Request 3: should have been blocked")
	}

}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := NewRateLimiter()
	address := "2.2.2.2"
	reqWindow := 10
	limit := 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if limiter.Allow(address, reqWindow, limit) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}

		}()
	}
	wg.Wait()

	if successCount != limit {
		t.Errorf("expected %d successful requests, got %d", limit, successCount)
	}

}
