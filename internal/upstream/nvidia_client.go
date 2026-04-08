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
)

var ErrUnexpectedUpstream = errors.New("unexpected upstream response")

type NVIDIAClient struct {
	httpClient http.Client
	baseURL    string
	apiKey     string
}

func NewNVIDIAClient(client http.Client, baseURL, apiKey string) *NVIDIAClient {
	return &NVIDIAClient{
		httpClient: client,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
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
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return RecognizeResult{}, http.StatusInternalServerError, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return RecognizeResult{}, http.StatusInternalServerError, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return RecognizeResult{}, http.StatusGatewayTimeout, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return RecognizeResult{}, http.StatusBadGateway, err
	}

	if resp.StatusCode >= 400 {
		return RecognizeResult{}, resp.StatusCode, errors.New(strings.TrimSpace(string(respBody)))
	}

	var parsed ChatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return RecognizeResult{}, http.StatusBadGateway, err
	}
	if len(parsed.Choices) == 0 {
		return RecognizeResult{}, http.StatusBadGateway, ErrUnexpectedUpstream
	}

	return RecognizeResult{
		Text: parsed.Choices[0].Message.Content,
	}, http.StatusOK, nil
}

func fileToDataURL(path, mimeType string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}
