// Package audit is the Enterprise Audit log (ADR 0022): the subscriber that persists
// `audit.entry` domain events into the append-only audit_entries table, and the read
// side behind the /site page. It runs only under a license that includes the audit
// feature; without one nothing is stored and nothing is readable.
//
// Governed by ee/LICENSE, not the AGPL.
package audit

import (
	"context"
	"fmt"

	"github.com/mokevnin/1mail/ee/licensekey"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/auditentry"
	"github.com/mokevnin/1mail/internal/events"
)

// ConsumerName names the bus consumer group of the audit subscriber.
const ConsumerName = "audit"

// Consumer is the audit subscriber. It is the raw-client entry for audit entries (an
// event envelope carries only a Workspace id, ADR 0017). It persists `audit.entry`
// events, deduplicated by the envelope id so at-least-once redelivery writes one row,
// and ignores every other event. Unlicensed, it consumes and discards.
func Consumer(client *ent.Client, lic *licensekey.License) events.Consumer {
	return events.Consumer{
		Name: ConsumerName,
		Handle: func(ctx context.Context, env events.Envelope) error {
			if env.Name != events.NameAuditEntry || !lic.Has(licensekey.FeatureAudit) {
				return nil
			}
			ev, err := events.Decode(env)
			if err != nil {
				return err
			}
			entry, ok := ev.(*events.AuditEntry)
			if !ok {
				return nil
			}
			return persist(ctx, client, env, entry)
		},
	}
}

func persist(ctx context.Context, client *ent.Client, env events.Envelope, e *events.AuditEntry) error {
	create := client.Scoped(env.WorkspaceID).AuditEntry().Create().
		SetEntryKey(env.ID).
		SetOccurredAt(env.OccurredAt).
		SetActorKind(e.Actor.Kind).
		SetAction(e.Action).
		SetTargetType(e.TargetType)
	if e.Actor.ID != "" {
		create.SetActorID(e.Actor.ID)
	}
	if e.Actor.Name != "" {
		create.SetActorName(e.Actor.Name)
	}
	if e.TargetID != "" {
		create.SetTargetID(e.TargetID)
	}
	if e.TargetName != "" {
		create.SetTargetName(e.TargetName)
	}
	if len(e.Diff) > 0 {
		create.SetDiff(e.Diff)
	}
	if e.RequestID != "" {
		create.SetRequestID(e.RequestID)
	}
	if e.IP != "" {
		create.SetIP(e.IP)
	}
	if e.UserAgent != "" {
		create.SetUserAgent(e.UserAgent)
	}
	if err := create.
		OnConflictColumns(auditentry.FieldWorkspaceID, auditentry.FieldEntryKey).
		Ignore().
		Exec(ctx); err != nil {
		return fmt.Errorf("persist audit entry %q: %w", env.ID, err)
	}
	return nil
}

// Log is the read side of the Audit log. Its zero value is unlicensed.
type Log struct {
	lic *licensekey.License
}

// NewLog builds the reader under a license.
func NewLog(lic *licensekey.License) *Log { return &Log{lic: lic} }

// Licensed reports whether this instance may show an Audit log.
func (l *Log) Licensed() bool { return l != nil && l.lic.Has(licensekey.FeatureAudit) }

// Entries returns up to limit entries of the scoped Workspace, newest first, that
// precede the cursor (an entry id; 0 starts from the newest), and the cursor of the
// next page (0 on the last page). The caller has already checked Licensed and the
// reader's role.
func (l *Log) Entries(ctx context.Context, s *ent.Scoped, cursor int64, limit int) ([]*ent.AuditEntry, int64, error) {
	q := s.AuditEntry().Query().Order(ent.Desc(auditentry.FieldID)).Limit(limit + 1)
	if cursor > 0 {
		q = q.Where(auditentry.IDLT(cursor))
	}
	rows, err := q.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) <= limit {
		return rows, 0, nil
	}
	rows = rows[:limit]
	return rows, rows[limit-1].ID, nil
}
