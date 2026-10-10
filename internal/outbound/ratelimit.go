package outbound

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/sendlimit"
)

// reserve takes one token from the default Integration's Send rate limit (ADR 0023).
// It returns the wait until capacity returns when the limit is spent, and zero when
// the message may go (also when the Integration is not limited, which costs one
// read and no write). Transactional is never delayed by the limit: it is not
// reserved here.
func (m *Module) reserve(ctx context.Context, s *ent.Scoped, req Request) (time.Duration, error) {
	if req.Kind == outboundmessage.KindTransactional {
		return 0, nil
	}
	integ, err := messaging.DefaultEmailIntegration(ctx, s)
	if errors.Is(err, messaging.ErrNoProvider) {
		return 0, nil // the gate already resolved a sender; a test double has no row
	}
	if err != nil {
		return 0, fmt.Errorf("outbound: load integration: %w", err)
	}
	limits := sendlimit.Limits{PerSecond: integ.MaxPerSecond, PerDay: integ.MaxPerDay}
	wait, err := sendlimit.Reserve(ctx, s, integ.ID, limits, m.now())
	if err != nil {
		return 0, fmt.Errorf("outbound: reserve send capacity: %w", err)
	}
	return wait, nil
}
