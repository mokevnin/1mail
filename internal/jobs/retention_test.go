package jobs

import (
	"log/slog"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
)

func TestRiverConfigSetsExplicitJobRetention(t *testing.T) {
	cfg := newRiverConfig(river.NewWorkers(), slog.Default())

	assert.Equal(t, 24*time.Hour, cfg.CompletedJobRetentionPeriod)
	assert.Equal(t, 24*time.Hour, cfg.CancelledJobRetentionPeriod)
	assert.Equal(t, 14*24*time.Hour, cfg.DiscardedJobRetentionPeriod)
}
