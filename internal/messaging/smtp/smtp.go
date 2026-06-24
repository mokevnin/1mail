// Package smtp implements the SMTP email provider on top of nikoksr/notify's
// mail service (which wraps net/smtp + STARTTLS). The native SES provider lives
// in the sibling ses package.
package smtp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nikoksr/notify/service/mail"

	"github.com/mokevnin/1mail/internal/messaging"
)

// Config is the cleartext credential shape stored (encrypted) for an SMTP
// integration. Password is the only secret; it is never echoed back by the API.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	From     string `json:"from"`
	FromName string `json:"fromName,omitempty"`
}

// Descriptor registers the SMTP provider with the messaging catalog.
func Descriptor() messaging.ProviderDescriptor {
	return messaging.ProviderDescriptor{
		Channel:  messaging.ChannelEmail,
		Provider: messaging.ProviderSMTP,
		Validate: validate,
		Build:    build,
	}
}

func validate(raw []byte) error {
	cfg, err := parse(raw)
	if err != nil {
		return err
	}
	switch {
	case cfg.Host == "":
		return fmt.Errorf("smtp: host is required")
	case cfg.Port <= 0:
		return fmt.Errorf("smtp: port must be positive")
	case cfg.From == "":
		return fmt.Errorf("smtp: from is required")
	}
	return nil
}

func build(raw []byte) (any, error) {
	cfg, err := parse(raw)
	if err != nil {
		return nil, err
	}
	return &sender{cfg: cfg}, nil
}

func parse(raw []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("smtp: invalid config: %w", err)
	}
	return cfg, nil
}

type sender struct {
	cfg Config
}

func (s *sender) Send(ctx context.Context, msg messaging.EmailMessage) error {
	from := messaging.FirstNonEmpty(msg.From, s.cfg.From)
	fromName := messaging.FirstNonEmpty(msg.FromName, s.cfg.FromName)

	m := mail.New(messaging.FormatSender(from, fromName), fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port))
	// Authenticate whenever a credential is set; an empty pair means an open relay.
	if s.cfg.Username != "" || s.cfg.Password != "" {
		m.AuthenticateSMTP("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	m.AddReceivers(msg.To)

	body := msg.Text
	if msg.HTML != "" {
		m.BodyFormat(mail.HTML)
		body = msg.HTML
	} else {
		m.BodyFormat(mail.PlainText)
	}
	return m.Send(ctx, msg.Subject, body)
}
