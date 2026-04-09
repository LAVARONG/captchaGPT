package upstream

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var ErrUnexpectedUpstream = errors.New("unexpected upstream response")

type NVIDIAClient struct {
	httpClient      http.Client
	baseURL         string
	apiKey          string
	enableThinking bool
}

func NewNVIDIAClient(client http.Client, baseURL, apiKey string, enableThinking bool) *NVIDIAClient {
	return &NVIDIAClient{
		httpClient:      client,
		baseURL:         strings.TrimRight(baseURL, "/"),
		apiKey:          apiKey,
		enableThinking: enableThinking,
	}
}

func (c *NVIDIAClient) RecognizeCaptcha(ctx context.Context, req RecognizeRequest) (RecognizeResult, int, error) {
	imageDataURL, err := fileToDataURL(req.ImagePath, req.MIMEType)
	if err != nil {
		return RecognizeResult{}, http.StatusBadRequest, err
	}

	payload := ChatCompletionRequest{
		Model: req.Model,
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentPart{
					{Type: "text", Text: req.Prompt},
					{Type: "image_url", ImageURL: &ImageURL{URL: imageDataURL}},
				},
			},
		},
		Temperature: 0.1,
		ChatTemplateKwargs: ChatTemplateKwargs{
			EnableThinking: c.enableThinking,
		},
	}

	parsed, statusCode, err := c.doChatCompletion(ctx, payload)
	if err != nil {
		return RecognizeResult{}, statusCode, err
	}

	return RecognizeResult{
		Text: parsed.Choices[0].Message.Content,
	}, http.StatusOK, nil
}

func (c *NVIDIAClient) SelfTest(ctx context.Context, model string) (SelfTestResult, error) {
	payload := ChatCompletionRequest{
		Model: model,
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentPart{
					{Type: "text", Text: "你好，请简短回复一句问候语。"},
				},
			},
		},
		Temperature: 0.1,
		ChatTemplateKwargs: ChatTemplateKwargs{
			EnableThinking: c.enableThinking,
		},
	}

	startedAt := time.Now()
	respBody, statusCode, err := c.doChatCompletion(ctx, payload)
	if err != nil {
		return SelfTestResult{
			DurationMS: time.Since(startedAt).Milliseconds(),
			StatusCode: statusCode,
		}, err
	}

	return SelfTestResult{
		Reply:      strings.TrimSpace(respBody.Choices[0].Message.Content),
		DurationMS: time.Since(startedAt).Milliseconds(),
		StatusCode: statusCode,
	}, nil
}

func fileToDataURL(path, mimeType string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func (c *NVIDIAClient) doChatCompletion(ctx context.Context, payload ChatCompletionRequest) (ChatCompletionResponse, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return ChatCompletionResponse{}, http.StatusInternalServerError, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResponse{}, http.StatusInternalServerError, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatCompletionResponse{}, http.StatusGatewayTimeout, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return ChatCompletionResponse{}, http.StatusBadGateway, err
	}

	if resp.StatusCode >= 400 {
		return ChatCompletionResponse{}, resp.StatusCode, errors.New(strings.TrimSpace(string(respBody)))
	}

	var parsed ChatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return ChatCompletionResponse{}, http.StatusBadGateway, err
	}
	if len(parsed.Choices) == 0 {
		return ChatCompletionResponse{}, http.StatusBadGateway, ErrUnexpectedUpstream
	}

	return parsed, http.StatusOK, nil
}
