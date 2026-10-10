package registry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mokevnin/sphericon/internal/messaging"
	"github.com/mokevnin/sphericon/internal/messaging/registry"
)

func TestDefaultRegistersBuiltInProviders(t *testing.T) {
	cat := registry.Default()
	for _, p := range []messaging.Provider{messaging.ProviderSMTP, messaging.ProviderSES} {
		_, ok := cat.Descriptor(messaging.ChannelEmail, p)
		assert.True(t, ok, "provider %s", p)
	}
	_, ok := cat.Descriptor(messaging.ChannelSMS, messaging.ProviderSMTP)
	assert.False(t, ok)
}
