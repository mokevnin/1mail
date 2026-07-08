package messaging

import "fmt"

// ProviderDescriptor is the extensibility seam: it teaches the catalog how to
// validate and build one provider. Build returns a channel-specific sender as
// `any`; callers type-assert to the channel interface (e.g. EmailSender).
type ProviderDescriptor struct {
	Channel  Channel
	Provider Provider

	// Validate runs semantic checks on a decrypted, structurally-valid config
	// blob (the typed API union already enforces shape and required fields).
	Validate func(config []byte) error

	// Build constructs the live sender from a decrypted config blob. The signer
	// (may be nil) is stored by the sender to DKIM-sign outbound mail per ADR 0010.
	Build func(config []byte, signer Signer) (any, error)
}

// Catalog holds the registered provider descriptors keyed by (channel, provider).
type Catalog struct {
	descriptors map[catalogKey]ProviderDescriptor
}

type catalogKey struct {
	channel  Channel
	provider Provider
}

// NewCatalog builds a catalog from the given descriptors.
func NewCatalog(descriptors ...ProviderDescriptor) *Catalog {
	c := &Catalog{descriptors: make(map[catalogKey]ProviderDescriptor, len(descriptors))}
	for _, d := range descriptors {
		c.descriptors[catalogKey{d.Channel, d.Provider}] = d
	}
	return c
}

// Descriptor looks up a registered provider; ok is false for unknown pairs.
func (c *Catalog) Descriptor(channel Channel, provider Provider) (ProviderDescriptor, bool) {
	d, ok := c.descriptors[catalogKey{channel, provider}]
	return d, ok
}

// ChannelOf reports which channel a provider belongs to; ok is false for an
// unregistered provider.
func (c *Catalog) ChannelOf(provider Provider) (Channel, bool) {
	for k := range c.descriptors {
		if k.provider == provider {
			return k.channel, true
		}
	}
	return "", false
}

// Validate runs the provider's semantic validation, or reports an unknown provider.
func (c *Catalog) Validate(channel Channel, provider Provider, config []byte) error {
	d, ok := c.Descriptor(channel, provider)
	if !ok {
		return fmt.Errorf("unknown provider %q for channel %q", provider, channel)
	}
	if d.Validate == nil {
		return nil
	}
	return d.Validate(config)
}

// BuildEmail builds an EmailSender for an email-channel provider. signer (may be
// nil) is passed to the provider for native DKIM signing.
func (c *Catalog) BuildEmail(provider Provider, config []byte, signer Signer) (EmailSender, error) {
	d, ok := c.Descriptor(ChannelEmail, provider)
	if !ok {
		return nil, fmt.Errorf("unknown email provider %q", provider)
	}
	built, err := d.Build(config, signer)
	if err != nil {
		return nil, err
	}
	sender, ok := built.(EmailSender)
	if !ok {
		return nil, fmt.Errorf("provider %q did not build an EmailSender", provider)
	}
	return sender, nil
}
