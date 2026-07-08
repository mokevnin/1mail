package messaging

import (
	"context"
	"fmt"

	"github.com/wneessen/go-mail"
)

// BuildMIME turns an EmailMessage into a go-mail *mail.Msg. It is the single
// MIME builder shared by every email provider (smtp sends it directly; ses
// serializes it to raw bytes for SendRawEmail), so message structure —
// multipart/alternative, headers, encoding — is identical across providers.
//
// From/FromName must already be resolved (msg-level value or provider default)
// by the caller. When both HTML and Text are present the message is
// multipart/alternative with text/plain first and text/html as the richer
// alternative (clients render the last part they understand). A single body is
// sent as-is.
func BuildMIME(msg EmailMessage) (*mail.Msg, error) {
	m := mail.NewMsg()

	if msg.FromName != "" {
		if err := m.FromFormat(msg.FromName, msg.From); err != nil {
			return nil, fmt.Errorf("messaging: invalid from address: %w", err)
		}
	} else if err := m.From(msg.From); err != nil {
		return nil, fmt.Errorf("messaging: invalid from address: %w", err)
	}

	if err := m.To(msg.To); err != nil {
		return nil, fmt.Errorf("messaging: invalid to address: %w", err)
	}
	m.Subject(msg.Subject)

	switch {
	case msg.HTML != "" && msg.Text != "":
		m.SetBodyString(mail.TypeTextPlain, msg.Text)
		m.AddAlternativeString(mail.TypeTextHTML, msg.HTML)
	case msg.HTML != "":
		m.SetBodyString(mail.TypeTextHTML, msg.HTML)
	default:
		m.SetBodyString(mail.TypeTextPlain, msg.Text)
	}

	return m, nil
}

// BuildSignedMIME builds the message via BuildMIME and attaches a native DKIM
// signer for msg.From's verified sending domain (ADR 0010). go-mail signs during
// WriteTo, so the same Msg signs identically whether the provider writes it
// directly (SES raw bytes) or through the SMTP DATA stream.
//
// The gate (slice 3): when signer is non-nil (a workspace send) and From's
// domain is not a verified sending domain, the send is rejected with
// ErrUnverifiedSendingDomain — a verified domain is required to send. A nil
// signer (platform/system mail) is exempt and always sends unsigned.
func BuildSignedMIME(ctx context.Context, msg EmailMessage, signer Signer) (*mail.Msg, error) {
	m, err := BuildMIME(msg)
	if err != nil {
		return nil, err
	}
	if signer == nil {
		return m, nil
	}
	dkim, err := signer.DKIMSigner(ctx, msg.From)
	if err != nil {
		return nil, err
	}
	if dkim == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnverifiedSendingDomain, msg.From)
	}
	m.SetDKIM(dkim)
	return m, nil
}
