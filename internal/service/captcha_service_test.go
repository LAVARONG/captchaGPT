package service

import "testing"

func TestSanitizeCaptcha(t *testing.T) {
	got := sanitizeCaptcha("Captcha: 7K3A\n")
	if got != "7K3A" {
		t.Fatalf("expected 7K3A, got %q", got)
	}
}
