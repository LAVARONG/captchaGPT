package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNVIDIAClientRecognizeCaptcha(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "sample.png")
	if err := os.WriteFile(tempFile, []byte("fake"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer nvidia-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","choices":[{"message":{"content":"7K3A"}}]}`))
	}))
	defer server.Close()

	client := NewNVIDIAClient(http.Client{}, server.URL, "nvidia-key")
	result, status, err := client.RecognizeCaptcha(context.Background(), RecognizeRequest{
		Model:     "google/gemma-4-31b-it",
		Prompt:    "return only the captcha text",
		ImagePath: tempFile,
		MIMEType:  "image/png",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if strings.TrimSpace(result.Text) != "7K3A" {
		t.Fatalf("unexpected result: %q", result.Text)
	}
}
