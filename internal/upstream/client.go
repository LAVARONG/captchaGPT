package upstream

import (
	"context"
	"captchagpt/internal/config"
	"errors"
	"net/http"
	"time"
)

type VisionClient interface {
	RecognizeCaptcha(ctx context.Context, req RecognizeRequest) (RecognizeResult, int, error)
}

func NewVisionClient(cfg config.Config) (VisionClient, error) {
	switch cfg.UpstreamProvider {
	case "nvidia", "":
		return NewNVIDIAClient(http.Client{
			Timeout: time.Duration(cfg.RequestTimeoutS) * time.Second,
		}, cfg.UpstreamBaseURL, cfg.NVIDIAAPIKey), nil
	default:
		return nil, errors.New("unsupported upstream provider: " + cfg.UpstreamProvider)
	}
}
