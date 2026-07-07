package jobs

import (
	"context"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/sendingdomain"
	"github.com/mokevnin/1mail/internal/sending"
)

// recheckBatchSize bounds how many domains one periodic tick re-checks, so a
// large workspace population can't stall the tick. Domains are picked
// least-recently-checked first, so every domain is reached over successive ticks.
const recheckBatchSize = 100

// --- on-demand single-domain verification (the "Verify" button) ---

type VerifySendingDomainArgs struct {
	SendingDomainID int64 `json:"sending_domain_id"`
}

func (VerifySendingDomainArgs) Kind() string { return "sending_domain_verify" }

type VerifySendingDomainWorker struct {
	river.WorkerDefaults[VerifySendingDomainArgs]
	ent    *ent.Client
	lookup sending.TXTLookup
}

func (w *VerifySendingDomainWorker) Work(ctx context.Context, job *river.Job[VerifySendingDomainArgs]) error {
	_, err := VerifySendingDomainByID(ctx, w.ent, w.lookup, job.Args.SendingDomainID)
	return err
}

// --- periodic re-check of all domains (verified is a live property, ADR 0010) ---

type RecheckSendingDomainsArgs struct{}

func (RecheckSendingDomainsArgs) Kind() string { return "sending_domain_recheck" }

type RecheckSendingDomainsWorker struct {
	river.WorkerDefaults[RecheckSendingDomainsArgs]
	ent *ent.Client
}

// Work fans the periodic tick out into one VerifySendingDomain job per domain,
// giving each check its own isolation/retry and reusing the same path as the
// "Verify" button. Domains are picked least-recently-checked first with NULLS
// FIRST, so a freshly-added, never-checked domain (last_checked_at IS NULL) is
// verified soonest rather than starved behind already-checked ones (plain ASC
// sorts NULLs last in Postgres).
func (w *RecheckSendingDomainsWorker) Work(ctx context.Context, job *river.Job[RecheckSendingDomainsArgs]) error {
	ids, err := DomainsDueForRecheck(ctx, w.ent, recheckBatchSize)
	if err != nil {
		return err
	}
	rc := river.ClientFromContext[pgx.Tx](ctx)
	for _, id := range ids {
		if _, err := rc.Insert(ctx, VerifySendingDomainArgs{SendingDomainID: id}, nil); err != nil {
			return err
		}
	}
	return nil
}

// DomainsDueForRecheck returns up to limit sending-domain ids to re-verify,
// least-recently-checked first with NULLS FIRST — never-checked domains come
// before already-checked ones. Extracted (and pure of the queue) so the ordering
// is testable without a river runtime.
func DomainsDueForRecheck(ctx context.Context, client *ent.Client, limit int) ([]int64, error) {
	return client.SendingDomain.Query().
		Order(sendingdomain.ByLastCheckedAt(sql.OrderAsc(), sql.OrderNullsFirst())).
		Limit(limit).
		IDs(ctx)
}

// VerifySendingDomainByID re-checks a domain's DKIM DNS and persists the result:
// verified, last_checked_at, and (when it becomes verified) verified_at. It is
// pure of the queue so it can be tested directly, and returns the resulting
// verified state.
//
// A verified→unverified flip is recorded here (structured log). ADR 0010 also
// calls for notifying the owner "the instant it flips" — that email is
// deliberately deferred to the send-gate slice (Slice 3): until sending is gated
// on a verified domain, an unverified domain blocks nothing, and the
// suspension-style notify pattern it mirrors (ADR 0007) is not built yet.
func VerifySendingDomainByID(ctx context.Context, client *ent.Client, lookup sending.TXTLookup, id int64) (bool, error) {
	dom, err := client.SendingDomain.Get(ctx, id)
	if err != nil {
		return false, err
	}

	verified, err := sending.VerifyDKIM(ctx, lookup, dom.DkimSelector, dom.Domain, dom.DkimPublicKey)
	if err != nil {
		return false, err
	}

	now := time.Now()
	upd := client.SendingDomain.UpdateOneID(id).SetLastCheckedAt(now)
	if verified != dom.Verified {
		upd.SetVerified(verified)
		if verified {
			upd.SetVerifiedAt(now)
		}
		if dom.Verified && !verified {
			slog.WarnContext(ctx, "sending domain DKIM verification lost",
				"sending_domain_id", id, "workspace_id", dom.WorkspaceID, "domain", dom.Domain)
		}
	}
	if err := upd.Exec(ctx); err != nil {
		return false, err
	}
	return verified, nil
}
