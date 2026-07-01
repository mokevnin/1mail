package service

import (
	"crypto/sha256"
	"encoding/hex"
)

const inviteTokenLen = 48

// GenerateInviteToken returns a random, URL-safe invitation token — the raw value
// that travels in the invite link and is never stored.
func GenerateInviteToken() (string, error) {
	return randomString(urlSafeAlphabet, inviteTokenLen)
}

// HashInviteToken returns the deterministic SHA-256 hex of a raw invite token.
// It is what the Invitation stores, so a presented token can be looked up by
// hash. SHA-256 (not bcrypt) is right here: the token is high-entropy random, so
// no slow hashing is needed and deterministic lookup is required.
func HashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
