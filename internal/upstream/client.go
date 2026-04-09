package upstream

import (
	"context"
	"captchagpt/internal/config"
	"errors"
	"net/http"
	"time"
)

type SelfTestResult struct {
	Reply      string
	DurationMS int64
	StatusCode int
}

type VisionClient interface {
	RecognizeCaptcha(ctx context.Context, req RecognizeRequest) (RecognizeResult, int, error)
	SelfTest(ctx context.Context, model string) (SelfTestResult, error)
}

func NewVisionClient(cfg config.Config) (VisionClient, error) {
	switch cfg.UpstreamProvider {
	case "nvidia", "":
		return NewNVIDIAClient(http.Client{
			Timeout: time.Duration(cfg.RequestTimeoutS) * time.Second,
		}, cfg.UpstreamBaseURL, cfg.NVIDIAAPIKey, cfg.EnableThinking), nil
	default:
		return nil, errors.New("unsupported upstream provider: " + cfg.UpstreamProvider)
	}
}
