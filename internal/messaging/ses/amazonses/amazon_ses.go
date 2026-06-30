// Package amazonses is a fork of github.com/nikoksr/notify/service/amazonses
// (v1.5.0), patched to accept a custom SES-compatible API endpoint so the same
// provider can target AWS SES or an SES-compatible service such as Yandex Cloud
// Postbox. notify's own amazonses exposes no way to override the endpoint (the
// ses.Client is unexported with no options), hence the fork.
//
// The only addition over upstream is the WithEndpoint option, which sets
// ses.Options.BaseEndpoint. Everything else mirrors upstream so the wrapper
// stays a thin, recognizable shim over aws-sdk-go-v2's ses client.
//
// Upstream is MIT-licensed:
//
//	Copyright (c) 2021 Nikos Kotseridis
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files, subject to the MIT
// License terms (see https://github.com/nikoksr/notify/blob/main/LICENSE).
package amazonses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type sesClient interface {
	SendEmail(
		ctx context.Context,
		params *ses.SendEmailInput,
		optFns ...func(options *ses.Options),
	) (*ses.SendEmailOutput, error)
}

// Compile-time check to ensure that ses.Client implements the sesClient interface.
var _ sesClient = new(ses.Client)

// Option customizes the AmazonSES service at construction.
type Option func(*options)

type options struct {
	endpoint string
}

// WithEndpoint overrides the SES API endpoint, targeting an SES-compatible
// service (e.g. Yandex Cloud Postbox) instead of AWS. Empty means AWS default.
func WithEndpoint(endpoint string) Option {
	return func(o *options) { o.endpoint = endpoint }
}

// AmazonSES struct holds necessary data to communicate with the Amazon Simple Email Service API.
type AmazonSES struct {
	client            sesClient
	senderAddress     *string
	receiverAddresses []string
}

// New returns a new instance of an AmazonSES notification service.
// You will need an Amazon Simple Email Service API access key and secret.
// See https://aws.github.io/aws-sdk-go-v2/docs/getting-started/
func New(accessKeyID, secretKey, region, senderAddress string, opts ...Option) (*AmazonSES, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	credProvider := credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, "")

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credProvider),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := ses.NewFromConfig(cfg, func(so *ses.Options) {
		if o.endpoint != "" {
			so.BaseEndpoint = aws.String(o.endpoint)
		}
	})

	return &AmazonSES{
		client:            client,
		senderAddress:     aws.String(senderAddress),
		receiverAddresses: []string{},
	}, nil
}

// AddReceivers takes email addresses and adds them to the internal address list. The Send method will send
// a given message to all those addresses.
func (a *AmazonSES) AddReceivers(addresses ...string) {
	a.receiverAddresses = append(a.receiverAddresses, addresses...)
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (a AmazonSES) Send(ctx context.Context, subject, message string) error {
	input := &ses.SendEmailInput{
		Source: a.senderAddress,
		Destination: &types.Destination{
			ToAddresses: a.receiverAddresses,
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(message),
				},
			},
			Subject: &types.Content{
				Data: aws.String(subject),
			},
		},
	}

	if _, err := a.client.SendEmail(ctx, input); err != nil {
		return fmt.Errorf("send mail using Amazon SES service: %w", err)
	}

	return nil
}
