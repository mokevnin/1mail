package messaging

import (
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
