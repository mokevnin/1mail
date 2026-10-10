package sendlimit

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/outboundmessage"
)

// Source says where an effective ceiling comes from. Only SourceManual is produced
// today; the provider-reported quota (SES GetSendQuota) joins it in Effective, which
// is the one place enforcement and display both read a ceiling from.
type Source string

const (
	// SourceManual is the operator-set value on the Integration.
	SourceManual Source = "manual"
	// SourceProvider is the value the provider reports.
	SourceProvider Source = "provider"
)

// Value is one effective ceiling: Limit is nil (and Source empty) when the window
// is not limited.
type Value struct {
	Limit  *int
	Source Source
}

// Effective is an Integration's ceilings as enforced: per second and per rolling
// 24 hours. Per ADR 0023 each is the minimum of the available values; when two
// sources give the same number the operator's manual value is reported.
type Effective struct {
	PerSecond Value
	PerDay    Value
}

// EffectiveOf computes the ceilings an Integration is held to.
func EffectiveOf(integ *ent.Integration) Effective {
	return Effective{
		PerSecond: manual(integ.MaxPerSecond),
		PerDay:    manual(integ.MaxPerDay),
	}
}

func manual(v *int) Value {
	if v == nil {
		return Value{}
	}
	return Value{Limit: v, Source: SourceManual}
}

// Limits is what the token buckets enforce.
func (e Effective) Limits() Limits {
	return Limits{PerSecond: e.PerSecond.Limit, PerDay: e.PerDay.Limit}
}

// Unlimited reports whether no ceiling applies at all.
func (e Effective) Unlimited() bool { return !e.Limits().Any() }

// window is the trailing period Usage counts over.
const window = 24 * time.Hour

// Usage counts the messages each of the Workspace's Integrations sent over the 24
// hours before now, keyed by Integration id (an Integration that sent nothing is
// absent). It counts every Outbound message the provider accepted, Transactional
// included, because that is what the provider's own quota sees.
func Usage(ctx context.Context, s *ent.Scoped, now time.Time) (map[int64]int, error) {
	var rows []struct {
		IntegrationID int64 `json:"integration_id"`
		Count         int   `json:"count"`
	}
	err := s.OutboundMessage().Query().
		Where(
			outboundmessage.StatusEQ(outboundmessage.StatusSent),
			outboundmessage.SentAtGT(now.Add(-window)),
			outboundmessage.IntegrationIDNotNil(),
		).
		GroupBy(outboundmessage.FieldIntegrationID).
		Aggregate(ent.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]int, len(rows))
	for _, r := range rows {
		out[r.IntegrationID] = r.Count
	}
	return out, nil
}
