package server

import "testing"

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(1, 1)
	if !limiter.Allow() {
		t.Fatal("first request should pass")
	}
	if limiter.Allow() {
		t.Fatal("second request should be limited")
	}
}
