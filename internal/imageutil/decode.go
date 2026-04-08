package imageutil

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"captchagpt/internal/requestid"
)

var (
	ErrInvalidImage     = errors.New("invalid image base64")
	ErrImageTooLarge    = errors.New("image too large")
	ErrUnsupportedImage = errors.New("unsupported image format")
)

type Meta struct {
	MIMEType  string
	SizeBytes int64
}

func DecodeAndSave(ctx context.Context, dir, encoded string, maxBytes int64) (string, Meta, error) {
	select {
	case <-ctx.Done():
		return "", Meta{}, ctx.Err()
	default:
	}

	raw, mimeType, err := normalizeAndDecode(encoded)
	if err != nil {
		return "", Meta{}, ErrInvalidImage
	}
	if int64(len(raw)) > maxBytes {
		return "", Meta{}, ErrImageTooLarge
	}

	format, err := detectImageFormat(raw)
	if err != nil {
		return "", Meta{}, ErrUnsupportedImage
	}
	if mimeType == "" {
		mimeType = mimeFromFormat(format)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", Meta{}, err
	}

	path := filepath.Join(dir, requestid.New("img_")+extensionFromFormat(format))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", Meta{}, err
	}

	return path, Meta{
		MIMEType:  mimeType,
		SizeBytes: int64(len(raw)),
	}, nil
}

func normalizeAndDecode(encoded string) ([]byte, string, error) {
	value := strings.TrimSpace(encoded)
	mimeType := ""

	if strings.HasPrefix(value, "data:") {
		comma := strings.Index(value, ",")
		if comma < 0 {
			return nil, "", ErrInvalidImage
		}
		header := value[:comma]
		value = value[comma+1:]
		semi := strings.Index(header, ";")
		if semi > len("data:") {
			mimeType = header[len("data:"):semi]
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, mimeType, nil
	}

	decoded, err = base64.RawStdEncoding.DecodeString(value)
	if err == nil {
		return decoded, mimeType, nil
	}

	return nil, "", err
}

func detectImageFormat(raw []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(raw))
	return format, err
}

func mimeFromFormat(format string) string {
	switch format {
	case "png":
		return "image/png"
	case "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func extensionFromFormat(format string) string {
	switch format {
	case "png":
		return ".png"
	case "jpeg":
		return ".jpg"
	case "gif":
		return ".gif"
	default:
		return ".bin"
	}
}
