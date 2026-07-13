package engine

import (
	"testing"
)

func TestRateLimiter_Sequential(t *testing.T) {
	newLimiter := NewRateLimiter()
	address := "1.1.1.1"
	reqWindow := 10
	limit := 2

	for i := range limit {
		if newLimiter.Allow(address, reqWindow, limit) == false {
			t.Errorf("Request %d: should not have been blocked", i+1)
		}
	}
	if newLimiter.Allow(address, reqWindow, limit) == true {
		t.Errorf("Request 3: should have been blocked")
	}

}
