package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"captchagpt/internal/api"
	"captchagpt/internal/imageutil"
	"captchagpt/internal/prompt"
	"captchagpt/internal/requestid"
	"captchagpt/internal/upstream"
)

type CaptchaService struct {
	modelName     string
	tempDir       string
	maxImageBytes int64
	visionClient  upstream.VisionClient
}

func New(modelName, tempDir string, maxImageBytes int64, client upstream.VisionClient) *CaptchaService {
	return &CaptchaService{
		modelName:     modelName,
		tempDir:       tempDir,
		maxImageBytes: maxImageBytes,
		visionClient:  client,
	}
}

func (s *CaptchaService) Recognize(ctx context.Context, req api.CaptchaRequest) (api.CaptchaResponse, int) {
	responseID := requestid.New("cap_")
	startedAt := time.Now()

	if strings.TrimSpace(req.ImageBase64) == "" {
		return api.NewErrorResponse(responseID, "missing_image_base64", "image_base64 is required", req.ClientRequestID), http.StatusBadRequest
	}

	path, meta, err := imageutil.DecodeAndSave(ctx, s.tempDir, req.ImageBase64, s.maxImageBytes)
	if err != nil {
		return errorResponseForDecode(responseID, s.modelName, req.ClientRequestID, err)
	}
	defer os.Remove(path)

	result, upstreamStatus, err := s.visionClient.RecognizeCaptcha(ctx, upstream.RecognizeRequest{
		Model:     s.modelName,
		Prompt:    prompt.Build(req.Captcha),
		ImagePath: path,
		MIMEType:  meta.MIMEType,
	})
	if err != nil {
		return errorResponseForUpstream(responseID, s.modelName, req.ClientRequestID, upstreamStatus, err)
	}

	text := sanitizeCaptcha(result.Text)
	if text == "" {
		return api.NewErrorResponse(responseID, "empty_model_response", "upstream model returned an empty result", req.ClientRequestID), http.StatusBadGateway
	}

	return api.NewSuccessResponse(responseID, text, time.Since(startedAt).Milliseconds()), http.StatusOK
}

func errorResponseForDecode(id, _model, reqID string, err error) (api.CaptchaResponse, int) {
	switch {
	case errors.Is(err, imageutil.ErrInvalidImage):
		return api.NewErrorResponse(id, "invalid_image_base64", "image_base64 must be a valid base64-encoded image", reqID), http.StatusBadRequest
	case errors.Is(err, imageutil.ErrImageTooLarge):
		return api.NewErrorResponse(id, "image_too_large", "image exceeds MAX_IMAGE_BYTES", reqID), http.StatusRequestEntityTooLarge
	case errors.Is(err, imageutil.ErrUnsupportedImage):
		return api.NewErrorResponse(id, "unsupported_image_format", "supported image formats are png, jpeg, gif", reqID), http.StatusUnsupportedMediaType
	default:
		return api.NewErrorResponse(id, "image_processing_failed", "failed to decode or save image", reqID), http.StatusInternalServerError
	}
}

func errorResponseForUpstream(id, _model, reqID string, upstreamStatus int, err error) (api.CaptchaResponse, int) {
	switch upstreamStatus {
	case http.StatusUnauthorized, http.StatusForbidden:
		return api.NewErrorResponse(id, "upstream_auth_failed", "upstream model authorization failed", reqID), http.StatusBadGateway
	case http.StatusTooManyRequests:
		return api.NewErrorResponse(id, "upstream_rate_limited", "upstream model rate limit exceeded", reqID), http.StatusServiceUnavailable
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return api.NewErrorResponse(id, "upstream_timeout", "upstream model request timed out", reqID), http.StatusGatewayTimeout
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return api.NewErrorResponse(id, "upstream_unavailable", "upstream model service is unavailable", reqID), http.StatusServiceUnavailable
	default:
		_ = err
		return api.NewErrorResponse(id, "upstream_request_failed", "failed to process request with upstream model", reqID), http.StatusBadGateway
	}
}

func sanitizeCaptcha(input string) string {
	value := strings.TrimSpace(input)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.Trim(value, `"'`)
	if idx := strings.Index(value, ":"); idx > 0 && strings.Contains(strings.ToLower(value[:idx]), "captcha") {
		value = strings.TrimSpace(value[idx+1:])
	}
	value = strings.Trim(value, `"'`)
	return value
}
