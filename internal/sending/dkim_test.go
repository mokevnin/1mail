package sending

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestGenerateKeypairRoundTrip(t *testing.T) {
	privPEM, pubTXT, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	if !strings.HasPrefix(pubTXT, "v=DKIM1; k=rsa; p=") {
		t.Fatalf("unexpected TXT value: %q", pubTXT)
	}
	if !strings.Contains(string(privPEM), "PRIVATE KEY") {
		t.Fatalf("private PEM missing key block: %q", privPEM)
	}

	// The public key re-derived from the stored private PEM must match the TXT
	// value handed to the user — this is exactly what verification compares.
	derived, err := PublicKeyFromPrivatePEM(privPEM)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivatePEM: %v", err)
	}
	if derived != pubTXT {
		t.Fatalf("derived public key mismatch:\n got %q\nwant %q", derived, pubTXT)
	}
}

func TestDKIMRecordHost(t *testing.T) {
	host, value := DKIMRecord("1mail", "mail.acme.com", "v=DKIM1; k=rsa; p=abc")
	if host != "1mail._domainkey.mail.acme.com" {
		t.Fatalf("host = %q", host)
	}
	if value != "v=DKIM1; k=rsa; p=abc" {
		t.Fatalf("value = %q", value)
	}
}

func stubLookup(records []string, err error) TXTLookup {
	return func(_ context.Context, _ string) ([]string, error) {
		return records, err
	}
}

func TestVerifyDKIM(t *testing.T) {
	const pub = "v=DKIM1; k=rsa; p=MIIBIjANBgkqABC"

	tests := []struct {
		name    string
		lookup  TXTLookup
		want    bool
		wantErr bool
	}{
		{
			name:   "exact match",
			lookup: stubLookup([]string{pub}, nil),
			want:   true,
		},
		{
			name:   "match among several records",
			lookup: stubLookup([]string{"v=spf1 -all", pub}, nil),
			want:   true,
		},
		{
			name:   "match despite interior whitespace in p=",
			lookup: stubLookup([]string{"v=DKIM1; k=rsa; p=MIIBIjANBg kqABC"}, nil),
			want:   true,
		},
		{
			name:   "wrong key",
			lookup: stubLookup([]string{"v=DKIM1; k=rsa; p=DIFFERENT"}, nil),
			want:   false,
		},
		{
			name:   "no record published (NXDOMAIN)",
			lookup: stubLookup(nil, &net.DNSError{Err: "no such host", IsNotFound: true}),
			want:   false,
		},
		{
			name:    "resolver failure surfaces",
			lookup:  stubLookup(nil, &net.DNSError{Err: "server misbehaving", IsTemporary: true}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifyDKIM(context.Background(), tt.lookup, "1mail", "mail.acme.com", pub)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyDKIM: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
