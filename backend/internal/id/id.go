package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a prefixed identifier such as "geo_1a2b3c4d5e".
func New(prefix string) (string, error) {
	buf := make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}
