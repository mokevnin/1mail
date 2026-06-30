// Package ses implements the Amazon SES email provider on top of nikoksr/notify's
// amazonses service.
package ses

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/ses/amazonses"
)

// Config is the cleartext credential shape stored (encrypted) for an SES
// integration. SecretAccessKey is the only secret; it is never echoed back.
type Config struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	From            string `json:"from"`
	FromName        string `json:"fromName,omitempty"`
	// Endpoint targets an SES-compatible API (e.g. Yandex Cloud Postbox) instead
	// of AWS. Empty ⇒ AWS default endpoint for Region. Not a secret.
	Endpoint string `json:"endpoint,omitempty"`
}

// Descriptor registers the SES provider with the messaging catalog.
func Descriptor() messaging.ProviderDescriptor {
	return messaging.ProviderDescriptor{
		Channel:  messaging.ChannelEmail,
		Provider: messaging.ProviderSES,
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
	case cfg.Region == "":
		return fmt.Errorf("ses: region is required")
	case cfg.AccessKeyID == "":
		return fmt.Errorf("ses: accessKeyId is required")
	case cfg.From == "":
		return fmt.Errorf("ses: from is required")
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
		return Config{}, fmt.Errorf("ses: invalid config: %w", err)
	}
	return cfg, nil
}

type sender struct {
	cfg Config
}

func (s *sender) Send(ctx context.Context, msg messaging.EmailMessage) error {
	from := messaging.FirstNonEmpty(msg.From, s.cfg.From)
	fromName := messaging.FirstNonEmpty(msg.FromName, s.cfg.FromName)

	svc, err := amazonses.New(
		s.cfg.AccessKeyID, s.cfg.SecretAccessKey, s.cfg.Region, messaging.FormatSender(from, fromName),
		amazonses.WithEndpoint(s.cfg.Endpoint),
	)
	if err != nil {
		return err
	}
	svc.AddReceivers(msg.To)

	body := msg.Text
	if msg.HTML != "" {
		body = msg.HTML
	}
	return svc.Send(ctx, msg.Subject, body)
}
