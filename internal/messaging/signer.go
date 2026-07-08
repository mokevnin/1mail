package messaging

import (
	"context"
	"fmt"
	"strings"

	"github.com/wneessen/go-mail"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/sendingdomain"
	"github.com/mokevnin/1mail/internal/secrets"
)

// dkimSigner is the workspace-bound Signer: it maps an outbound From address to
// the workspace's verified sending domain and builds a native go-mail DKIM
// signer from the domain's Tink-encrypted private key (ADR 0010). Bound to one
// workspace so every lookup is workspace-scoped.
type dkimSigner struct {
	ent         *ent.Client
	cipher      *secrets.Cipher
	workspaceID int64
}

// NewDKIMSigner builds a Signer scoped to workspaceID.
func NewDKIMSigner(client *ent.Client, cipher *secrets.Cipher, workspaceID int64) Signer {
	return &dkimSigner{ent: client, cipher: cipher, workspaceID: workspaceID}
}

// DKIMSigner returns a signer for fromEmail's domain when that domain is a
// verified sending domain in this workspace, or (nil, nil) otherwise — an
// unverified or unknown From is left unsigned here (the send gate is slice 3).
// The match is on the exact From domain; a subdomain of a verified domain does
// not match and is left unsigned.
func (s *dkimSigner) DKIMSigner(ctx context.Context, fromEmail string) (*mail.DKIMSigner, error) {
	domain := domainOf(fromEmail)
	if domain == "" {
		return nil, nil
	}

	row, err := s.ent.SendingDomain.Query().
		Where(
			sendingdomain.WorkspaceID(s.workspaceID),
			sendingdomain.Domain(domain),
			sendingdomain.Verified(true),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	privPEM, err := s.cipher.Decrypt(row.DkimPrivateKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("messaging: decrypt dkim key for %s: %w", domain, err)
	}
	key, err := mail.PrivKeyFromPEM(privPEM)
	if err != nil {
		return nil, fmt.Errorf("messaging: parse dkim key for %s: %w", domain, err)
	}
	return mail.NewDKIMSigner(row.Domain, row.DkimSelector, key), nil
}

// domainOf returns the lower-cased domain of an email address, or "" when there
// is no single "@".
func domainOf(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}
