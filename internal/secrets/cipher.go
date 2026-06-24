// Package secrets provides authenticated encryption for credentials stored at
// rest (e.g. workspace provider configs), backed by Google Tink. Tink ciphertext
// carries the key id, so key rotation is native (a keyset can hold multiple keys
// with one primary for encryption). The keyset is loaded from config as a
// base64-encoded cleartext keyset today; swapping in a KMS-wrapped keyset later
// is a load-time change that leaves Encrypt/Decrypt callers untouched.
package secrets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

// ErrCiphertext is returned when a stored value cannot be decrypted.
var ErrCiphertext = errors.New("invalid or corrupt ciphertext")

// Cipher encrypts and decrypts secret values with a Tink AEAD primitive.
type Cipher struct {
	aead tink.AEAD
}

// NewCipher builds a Cipher from a base64-encoded Tink keyset. It fails fast
// (caller wires this at boot) when the keyset is missing or malformed.
func NewCipher(keysetBase64 string) (*Cipher, error) {
	if strings.TrimSpace(keysetBase64) == "" {
		return nil, errors.New("encryption keyset is required")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keysetBase64))
	if err != nil {
		return nil, fmt.Errorf("decode encryption keyset: %w", err)
	}
	kh, err := insecurecleartextkeyset.Read(keyset.NewBinaryReader(bytes.NewReader(raw)))
	if err != nil {
		return nil, fmt.Errorf("read encryption keyset: %w", err)
	}
	a, err := aead.New(kh)
	if err != nil {
		return nil, fmt.Errorf("build AEAD: %w", err)
	}
	return &Cipher{aead: a}, nil
}

// Encrypt seals plaintext and returns a base64 string safe to store in a text column.
func (c *Cipher) Encrypt(plaintext []byte) (string, error) {
	ct, err := c.aead.Encrypt(plaintext, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(value string) ([]byte, error) {
	ct, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCiphertext, err)
	}
	pt, err := c.aead.Decrypt(ct, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCiphertext, err)
	}
	return pt, nil
}

// GenerateKeysetBase64 mints a fresh AES-256-GCM keyset and returns it
// base64-encoded — for seeding env config and tests.
func GenerateKeysetBase64() (string, error) {
	kh, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := insecurecleartextkeyset.Write(kh, keyset.NewBinaryWriter(buf)); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
