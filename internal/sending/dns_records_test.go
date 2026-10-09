package sending

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

func TestSPFAndDMARCRecords(t *testing.T) {
	host, value := SPFRecord("mail.acme.com")
	if host != "mail.acme.com" || value != "v=spf1 include:amazonses.com ~all" {
		t.Fatalf("SPFRecord = %q, %q", host, value)
	}

	host, value = DMARCRecord("mail.acme.com")
	if host != "_dmarc.mail.acme.com" || value != "v=DMARC1; p=none;" {
		t.Fatalf("DMARCRecord = %q, %q", host, value)
	}
}

func TestPublicKeyFromPrivatePEMRejectsBadKeys(t *testing.T) {
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	edDER, err := x509.MarshalPKCS8PrivateKey(edKey)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		pem  []byte
		want string
	}{
		"no pem block":   {[]byte("not a pem"), "no PEM block"},
		"bad der":        {pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("junk")}), "parse private key"},
		"not an rsa key": {pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: edDER}), "not RSA"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := PublicKeyFromPrivatePEM(tc.pem)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
			if got != "" {
				t.Fatalf("got %q, want empty", got)
			}
		})
	}
}

func TestVerifyDKIMRejectsEmptyExpectedKey(t *testing.T) {
	lookup := func(context.Context, string) ([]string, error) { return []string{"v=DKIM1; k=rsa; p=abc"}, nil }
	ok, err := VerifyDKIM(context.Background(), lookup, "1mail", "acme.com", "v=DKIM1; k=rsa; p=")
	if ok || err == nil {
		t.Fatalf("VerifyDKIM = %v, %v; want false and an error for an empty expected key", ok, err)
	}
}
