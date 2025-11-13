package engine

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
)

type ChallengeRequest struct {
	Challenge string `json:"challenge"`
}

// BuildHMACMessage constructs the message string used to compute the HMAC.
func BuildHMACMessage(messageID, timestamp string, body []byte) []byte {
	msg := make([]byte, 0, len(messageID)+len(timestamp)+len(body))
	msg = append(msg, messageID...)
	msg = append(msg, timestamp...)
	msg = append(msg, body...)
	return msg
}

// ComputeHMAC returns the HMAC SHA-256 hex digest for the provided secret and message.
func ComputeHMAC(secret []byte, message []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(message)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// VerifyHMAC performs a constant-time comparison between the computed HMAC and the provided signature.
func VerifyHMAC(computedHex, receivedHex string) bool {
	if len(computedHex) != len(receivedHex) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computedHex), []byte(receivedHex)) == 1
}

// handleChallenge processes a challenge request and returns the challenge string.
func handleChallenge(body []byte) ([]byte, error) {
	var c ChallengeRequest
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	return []byte(c.Challenge), nil
}
