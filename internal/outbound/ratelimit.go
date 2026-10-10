package outbound

import (
	"context"
	"fmt"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/sendlimit"
)

// reserve takes one token from the default Integration's Send rate limit (ADR 0023).
// It returns the wait until capacity returns when the limit is spent, and zero when
// the message may go (also when the Integration is not limited, which costs one
// read and no write). Transactional is never delayed by the limit: it is not
// reserved here.
func (m *Module) reserve(ctx context.Context, s *ent.Scoped, req Request, integ *ent.Integration) (time.Duration, error) {
	if req.Kind == outboundmessage.KindTransactional || integ == nil {
		return 0, nil // an Integration-less sender (a test double) has no limit to spend
	}
	limits := sendlimit.EffectiveOf(integ).Limits()
	wait, err := sendlimit.Reserve(ctx, s, integ.ID, limits, m.now())
	if err != nil {
		return 0, fmt.Errorf("outbound: reserve send capacity: %w", err)
	}
	return wait, nil
}
