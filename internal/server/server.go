package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"captchagpt/internal/api"
	"captchagpt/internal/config"
	"captchagpt/internal/service"
	"captchagpt/internal/upstream"
)

type Server struct {
	cfg     config.Config
	httpSrv *http.Server
	service *service.CaptchaService
	limiter *RateLimiter
}

func New(cfg config.Config) (*http.Server, error) {
	client, err := upstream.NewVisionClient(cfg)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:     cfg,
		service: service.New(cfg.ModelName, cfg.TempDir, cfg.MaxImageBytes, client),
		limiter: NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/getCode", s.handleGetCode)

	s.httpSrv = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.recoverMiddleware(s.withMiddleware(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      time.Duration(cfg.RequestTimeoutS+15) * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s.httpSrv, nil
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleGetCode(w http.ResponseWriter, r *http.Request) {
	requestID := readOrCreateRequestID(r)

	var req api.CaptchaRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, s.cfg.MaxImageBytes*2)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope(s.cfg.ModelName, requestID, "invalid_json", "request body must be valid JSON"))
		return
	}

	resp, status := s.service.Recognize(r.Context(), req)
	if resp.Error != nil && resp.Error.RequestID == "" {
		resp.Error.RequestID = requestID
	}
	writeJSON(w, status, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func errorEnvelope(_model, requestID, code, message string) api.CaptchaResponse {
	return api.NewErrorResponse("cap_"+requestID, code, message, requestID)
}

func withTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
