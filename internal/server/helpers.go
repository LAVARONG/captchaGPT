package server

import (
	"net/http"
	"strings"

	"captchagpt/internal/requestid"
)

func readOrCreateRequestID(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Request-ID")); v != "" {
		return v
	}
	return requestid.New("req_")
}
