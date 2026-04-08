package imageutil

import (
	"context"
	"os"
	"testing"
)

func TestDecodeAndSave_Invalid(t *testing.T) {
	_, _, err := DecodeAndSave(context.Background(), t.TempDir(), "not-base64", 1024)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodeAndSave_TooLarge(t *testing.T) {
	path := t.TempDir()
	_, _, err := DecodeAndSave(context.Background(), path, "ZmFrZQ==", 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("temp dir should still exist: %v", statErr)
	}
}
