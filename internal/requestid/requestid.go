package requestid

import (
	"crypto/rand"
	"encoding/hex"
)

func New(prefix string) string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return prefix + hex.EncodeToString(buf)
}
