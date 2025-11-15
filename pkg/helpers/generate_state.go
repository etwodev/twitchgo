package helpers

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateState creates a base64url-encoded random state.
func GenerateState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
