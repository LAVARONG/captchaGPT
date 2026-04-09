package api

import "time"

type CaptchaHints struct {
	Task          string   `json:"task,omitempty"`
	Length        int      `json:"length,omitempty"`
	Charset       string   `json:"charset,omitempty"`
	AllowedChars  string   `json:"allowed_chars,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	Language      string   `json:"language,omitempty"`
	ExtraRules    []string `json:"extra_rules,omitempty"`
}

type CaptchaRequest struct {
	ImageBase64     string       `json:"image_base64"`
	Captcha         CaptchaHints `json:"captcha"`
	ClientRequestID string       `json:"client_request_id,omitempty"`
}

type CaptchaResult struct {
	Text       string `json:"text,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
}

type CaptchaResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Result  *CaptchaResult `json:"result,omitempty"`
	Error   *APIError      `json:"error,omitempty"`
}

func NewSuccessResponse(id, text string, durationMS int64) CaptchaResponse {
	return CaptchaResponse{
		ID:      id,
		Object:  "captcha.result",
		Created: time.Now().Unix(),
		Result: &CaptchaResult{
			Text:       text,
			DurationMS: durationMS,
		},
	}
}

func NewErrorResponse(id, code, message, reqID string) CaptchaResponse {
	return CaptchaResponse{
		ID:      id,
		Object:  "error",
		Created: time.Now().Unix(),
		Error: &APIError{
			Code:      code,
			Message:   message,
			Type:      "invalid_request_error",
			RequestID: reqID,
		},
	}
}
