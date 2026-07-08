// Package ses implements the Amazon SES email provider. It builds the message
// with the shared messaging.BuildMIME (same MIME as the smtp provider) and
// hands the raw bytes to SES SendRawEmail, so any custom headers (List-Unsubscribe,
// DKIM later) survive intact — SendEmail's structured API would strip them.
//
// The client is built directly on aws-sdk-go-v2's ses client with an optional
// BaseEndpoint override so the same provider can target AWS SES or an
// SES-compatible service such as Yandex Cloud Postbox.
package ses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"

	"github.com/mokevnin/1mail/internal/messaging"
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

func build(raw []byte, signer messaging.Signer) (any, error) {
	cfg, err := parse(raw)
	if err != nil {
		return nil, err
	}
	return &sender{cfg: cfg, signer: signer}, nil
}

func parse(raw []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("ses: invalid config: %w", err)
	}
	return cfg, nil
}

type sender struct {
	cfg    Config
	signer messaging.Signer
}

func (s *sender) Send(ctx context.Context, msg messaging.EmailMessage) error {
	msg.From = messaging.FirstNonEmpty(msg.From, s.cfg.From)
	msg.FromName = messaging.FirstNonEmpty(msg.FromName, s.cfg.FromName)

	m, err := messaging.BuildSignedMIME(ctx, msg, s.signer)
	if err != nil {
		return err
	}
	var raw bytes.Buffer
	if _, err := m.WriteTo(&raw); err != nil {
		return fmt.Errorf("ses: render message: %w", err)
	}

	client, err := s.client(ctx)
	if err != nil {
		return err
	}
	_, err = client.SendRawEmail(ctx, &ses.SendRawEmailInput{
		// Source is the envelope sender; the MIME From header carries the display
		// name. It must match a verified SES identity.
		Source:       aws.String(msg.From),
		Destinations: []string{msg.To},
		RawMessage:   &types.RawMessage{Data: raw.Bytes()},
	})
	if err != nil {
		return fmt.Errorf("ses: send: %w", err)
	}
	return nil
}

func (s *sender) client(ctx context.Context) (*ses.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(s.cfg.AccessKeyID, s.cfg.SecretAccessKey, "")
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
		config.WithRegion(s.cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("ses: load config: %w", err)
	}
	return ses.NewFromConfig(cfg, func(o *ses.Options) {
		if s.cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(s.cfg.Endpoint)
		}
	}), nil
}
