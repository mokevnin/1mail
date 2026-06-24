// Package messaging is the channel-agnostic management layer for workspace
// sending providers. It defines the per-channel send contracts (EmailSender
// today; SmsSender later), a provider catalog that maps a (channel, provider)
// pair to its validation + construction logic, and a resolver that turns a
// workspace's stored integration into a ready-to-use sender.
//
// The split is deliberate: storage, encryption, CRUD, default-selection and the
// catalog are generic across channels, while the send interface is channel
// specific. Adding a provider = register a descriptor + an impl. Adding a
// channel = a new send interface + descriptors, reusing all of the above.
package messaging

import "context"

// Channel identifies a delivery medium. Values mirror the ent Integration.channel enum.
type Channel string

const (
	ChannelEmail Channel = "email"
	// ChannelSMS is reserved; no SMS provider is implemented yet.
	ChannelSMS Channel = "sms"
)

// Provider identifies a concrete provider implementation. Values mirror the ent
// Integration.provider enum.
type Provider string

const (
	ProviderSMTP Provider = "smtp"
	ProviderSES  Provider = "ses"
)

// EmailMessage is the channel-specific payload for email sends. HTML is used
// when set, otherwise Text (the provider libraries take a single body).
type EmailMessage struct {
	From     string
	FromName string
	To       string
	Subject  string
	HTML     string
	Text     string
}

// EmailSender is implemented by every email provider (smtp, ses, …).
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// FirstNonEmpty returns the first non-empty string, or "" if all are empty.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// FormatSender renders an email sender as "Name <addr>" when a display name is
// present, otherwise just the address.
func FormatSender(addr, name string) string {
	if name != "" {
		return name + " <" + addr + ">"
	}
	return addr
}
