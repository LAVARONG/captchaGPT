package server

import (
	"fmt"
	"net/http"
)

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := readOrCreateRequestID(r)
				writeJSON(w, http.StatusInternalServerError, errorEnvelope(s.cfg.ModelName, requestID, "internal_server_error", fmt.Sprintf("internal error: %v", rec)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
