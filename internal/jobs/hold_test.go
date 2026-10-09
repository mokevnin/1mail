package jobs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/internal/outbound"
)

func snoozeOf(t *testing.T, err error) *rivertype.JobSnoozeError {
	t.Helper()
	var s *rivertype.JobSnoozeError
	require.ErrorAs(t, err, &s)
	return s
}

// A hold and a live claim are both "try again later", never "spend an attempt": a
// long hold must not exhaust a recipient's retry budget, and an attempt that merely
// raced another must not be marked failed.
func TestDeferrableErrorsBecomeSnoozes(t *testing.T) {
	assert.Equal(t, holdRetryDelay, snoozeOf(t, snoozeIfDeferrable(&HeldError{Reason: outbound.HoldSuspended})).Duration)
	assert.Equal(t, inProgressRetryDelay,
		snoozeOf(t, snoozeIfDeferrable(fmt.Errorf("wrapped: %w", outbound.ErrInProgress))).Duration)
}

func TestOtherErrorsPassThroughUnchanged(t *testing.T) {
	boom := errors.New("smtp unavailable")
	assert.Same(t, boom, snoozeIfDeferrable(boom))
	assert.False(t, isDeferrable(boom))
	assert.True(t, isDeferrable(&HeldError{Reason: outbound.HoldNoIntegration}))
	assert.True(t, isDeferrable(outbound.ErrInProgress))
}
