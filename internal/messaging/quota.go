package messaging

import "context"

// Quota is what a provider reports about its own sending limits (ADR 0023). A nil
// field is a window the provider does not bound.
type Quota struct {
	PerSecond *int
	PerDay    *int
}

// QuotaReader is implemented by a sender whose provider can report its quota (SES
// GetSendQuota). Providers that cannot (SMTP) do not implement it and have no
// provider ceiling.
type QuotaReader interface {
	SendQuota(ctx context.Context) (Quota, error)
}
