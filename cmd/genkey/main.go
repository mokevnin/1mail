// Command genkey prints a fresh base64-encoded Tink keyset for ENCRYPTION_KEY.
// Use it when bootstrapping a local environment:
//
//	make gen-encryption-key   # then paste the value into .env
package main

import (
	"fmt"
	"log"

	"github.com/mokevnin/1mail/internal/secrets"
)

func main() {
	key, err := secrets.GenerateKeysetBase64()
	if err != nil {
		log.Fatalf("generate keyset: %v", err)
	}
	fmt.Println(key)
}
