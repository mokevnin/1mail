package service

import (
	"crypto/rand"
	"math/big"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

const (
	tokenPrefix    = "omtk"
	prefixAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	prefixLen      = 12
	secretLen      = 40
	bcryptCost     = 12
)

var tokenRe = regexp.MustCompile(`^omtk_([a-z0-9]+)_([A-Za-z0-9_-]+)$`)

const urlSafeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"

type ParsedToken struct {
	Prefix string
	Secret string
}

func GenerateTokenPrefix() (string, error) {
	return randomString(prefixAlphabet, prefixLen)
}

func GenerateTokenSecret() (string, error) {
	return randomString(urlSafeAlphabet, secretLen)
}

func TokenValue(prefix, secret string) string {
	return tokenPrefix + "_" + prefix + "_" + secret
}

const collectKeyPrefix = "omck"

// GenerateCollectKey returns a per-workspace collect write-key (x-collect-key).
func GenerateCollectKey() (string, error) {
	secret, err := randomString(urlSafeAlphabet, secretLen)
	if err != nil {
		return "", err
	}
	return collectKeyPrefix + "_" + secret, nil
}

func HashTokenSecret(secret string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func VerifyTokenSecret(secret, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}

func ParseToken(value string) *ParsedToken {
	m := tokenRe.FindStringSubmatch(value)
	if m == nil {
		return nil
	}
	return &ParsedToken{Prefix: m[1], Secret: m[2]}
}

func randomString(alphabet string, n int) (string, error) {
	b := make([]byte, n)
	alphaLen := big.NewInt(int64(len(alphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, alphaLen)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}
