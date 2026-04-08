package server

import (
	"crypto/subtle"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	rps        float64
	burst      float64
	lastRefill time.Time
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		tokens:     float64(burst),
		rps:        rps,
		burst:      float64(burst),
		lastRefill: time.Now(),
	}
}

func (l *RateLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.lastRefill = now
	l.tokens += elapsed * l.rps
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := readOrCreateRequestID(r)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		if !s.limiter.Allow() {
			writeJSON(w, http.StatusTooManyRequests, errorEnvelope(s.cfg.ModelName, requestID, "rate_limit_exceeded", "too many requests"))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, errorEnvelope(s.cfg.ModelName, requestID, "missing_authorization", "Authorization header is required"))
			return
		}
		const prefix = "Bearer "
		if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
			writeJSON(w, http.StatusUnauthorized, errorEnvelope(s.cfg.ModelName, requestID, "invalid_authorization_format", "use Authorization: Bearer <key>"))
			return
		}
		if subtle.ConstantTimeCompare([]byte(authHeader[len(prefix):]), []byte(s.cfg.UserAPIKey)) != 1 {
			writeJSON(w, http.StatusForbidden, errorEnvelope(s.cfg.ModelName, requestID, "invalid_api_key", "API key is invalid"))
			return
		}

		ctx, cancel := withTimeout(r.Context(), time.Duration(s.cfg.RequestTimeoutS)*time.Second)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
