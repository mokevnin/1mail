package site

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/automation"
	"github.com/mokevnin/1mail/ent/automationrun"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	siteapi "github.com/mokevnin/1mail/gen/site"
)

// analyticsDayFormat is the bucket label format for the engagement time series.
// Buckets are UTC days (date_trunc('day', …)); a known v1 limitation is that all
// workspaces see UTC-aligned days regardless of their locale.
const analyticsDayFormat = "2006-01-02"

// rangeDays maps the selectable analytics window to a number of days (inclusive
// of today). 30d is the default when the param is absent or unrecognized.
func rangeDays(r siteapi.OptSiteAnalyticsRange) int {
	if v, ok := r.Get(); ok {
		switch v {
		case siteapi.SiteAnalyticsRange7d:
			return 7
		case siteapi.SiteAnalyticsRange90d:
			return 90
		}
	}
	return 30
}

// SiteAnalyticsOverview returns aggregate metrics for the workspace dashboard.
// Engagement KPIs and the time series share a single range-filterable source
// (BroadcastRecipient delivery timestamps) so the cards and the chart reconcile;
// contact and automation counts are point-in-time snapshots.
func (h *Handlers) SiteAnalyticsOverview(ctx context.Context, params siteapi.SiteAnalyticsOverviewParams) (siteapi.SiteAnalyticsOverviewRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	days := rangeDays(params.Range)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	since := today.AddDate(0, 0, -(days - 1))
	until := today.AddDate(0, 0, 1) // exclusive upper bound (midnight tomorrow UTC)

	contacts, err := h.analyticsContacts(ctx, ws, since)
	if err != nil {
		return nil, err
	}
	email, err := h.analyticsEmail(ctx, ws, since, until)
	if err != nil {
		return nil, err
	}
	automations, err := h.analyticsAutomations(ctx, ws)
	if err != nil {
		return nil, err
	}
	series, err := h.analyticsTimeseries(ctx, ws, since, until)
	if err != nil {
		return nil, err
	}

	return &siteapi.SiteAnalyticsOverview{
		Contacts:    contacts,
		Email:       email,
		Automations: automations,
		Timeseries:  series,
	}, nil
}

func (h *Handlers) analyticsContacts(ctx context.Context, ws int64, since time.Time) (siteapi.SiteAnalyticsContacts, error) {
	var out siteapi.SiteAnalyticsContacts
	total, err := h.ent.Contact.Query().Where(contact.WorkspaceID(ws)).Count(ctx)
	if err != nil {
		return out, err
	}
	active, err := h.ent.Contact.Query().Where(contact.WorkspaceID(ws), contact.StatusEQ(contact.StatusActive)).Count(ctx)
	if err != nil {
		return out, err
	}
	unsub, err := h.ent.Contact.Query().Where(contact.WorkspaceID(ws), contact.StatusEQ(contact.StatusUnsubscribed)).Count(ctx)
	if err != nil {
		return out, err
	}
	newInRange, err := h.ent.Contact.Query().Where(contact.WorkspaceID(ws), contact.CreatedAtGTE(since)).Count(ctx)
	if err != nil {
		return out, err
	}
	return siteapi.SiteAnalyticsContacts{
		Total:        int32(total),
		Active:       int32(active),
		Unsubscribed: int32(unsub),
		NewInRange:   int32(newInRange),
	}, nil
}

func (h *Handlers) analyticsEmail(ctx context.Context, ws int64, since, until time.Time) (siteapi.SiteAnalyticsEmail, error) {
	var out siteapi.SiteAnalyticsEmail
	// Cohort by send: the denominator is the messages sent in the window, and
	// opens/clicks are counted among that same cohort. This keeps opened ≤ sent
	// (rates stay in [0,1]) and lets the KPIs reconcile with the time series,
	// which buckets the same cohort by send day.
	cohort := func() *ent.BroadcastRecipientQuery {
		return h.ent.BroadcastRecipient.Query().Where(
			broadcastrecipient.WorkspaceID(ws),
			broadcastrecipient.SentAtGTE(since),
			broadcastrecipient.SentAtLT(until),
		)
	}
	sent, err := cohort().Count(ctx)
	if err != nil {
		return out, err
	}
	opened, err := cohort().Where(broadcastrecipient.OpenedAtNotNil()).Count(ctx)
	if err != nil {
		return out, err
	}
	clicked, err := cohort().Where(broadcastrecipient.ClickedAtNotNil()).Count(ctx)
	if err != nil {
		return out, err
	}
	return siteapi.SiteAnalyticsEmail{
		SentCount:       int32(sent),
		OpenedCount:     int32(opened),
		ClickedCount:    int32(clicked),
		OpenRate:        ratio(opened, sent),
		ClickRate:       ratio(clicked, sent),
		ClickToOpenRate: ratio(clicked, opened),
	}, nil
}

func (h *Handlers) analyticsAutomations(ctx context.Context, ws int64) (siteapi.SiteAnalyticsAutomations, error) {
	var out siteapi.SiteAnalyticsAutomations
	total, err := h.ent.Automation.Query().Where(automation.WorkspaceID(ws)).Count(ctx)
	if err != nil {
		return out, err
	}
	active, err := h.ent.Automation.Query().Where(automation.WorkspaceID(ws), automation.StatusEQ(automation.StatusActive)).Count(ctx)
	if err != nil {
		return out, err
	}
	runsActive, err := h.ent.AutomationRun.Query().Where(automationrun.WorkspaceID(ws), automationrun.StatusEQ(automationrun.StatusActive)).Count(ctx)
	if err != nil {
		return out, err
	}
	runsCompleted, err := h.ent.AutomationRun.Query().Where(automationrun.WorkspaceID(ws), automationrun.StatusEQ(automationrun.StatusCompleted)).Count(ctx)
	if err != nil {
		return out, err
	}
	return siteapi.SiteAnalyticsAutomations{
		Total:         int32(total),
		Active:        int32(active),
		RunsActive:    int32(runsActive),
		RunsCompleted: int32(runsCompleted),
	}, nil
}

// analyticsTimeseries buckets the send cohort by UTC send day — sent, plus the
// opened/clicked subsets — then zero-fills every day in [since, until) so the
// chart has no gaps. date_trunc is forced to UTC (via AT TIME ZONE 'UTC') so the
// bucket labels match the UTC zero-fill loop regardless of the DB session zone.
func (h *Handlers) analyticsTimeseries(ctx context.Context, ws int64, since, until time.Time) ([]siteapi.SiteAnalyticsPoint, error) {
	var rows []struct {
		Day     string `sql:"day"`
		Sent    int    `sql:"sent"`
		Opened  int    `sql:"opened"`
		Clicked int    `sql:"clicked"`
	}
	err := h.ent.BroadcastRecipient.Query().
		Modify(func(s *sql.Selector) {
			sentAt := s.C(broadcastrecipient.FieldSentAt)
			s.Select(
				sql.As(fmt.Sprintf("to_char(date_trunc('day', %s AT TIME ZONE 'UTC'), 'YYYY-MM-DD')", sentAt), "day"),
				sql.As("COUNT(*)", "sent"),
				sql.As(fmt.Sprintf("COUNT(%s)", s.C(broadcastrecipient.FieldOpenedAt)), "opened"),
				sql.As(fmt.Sprintf("COUNT(%s)", s.C(broadcastrecipient.FieldClickedAt)), "clicked"),
			).
				Where(sql.And(
					sql.EQ(s.C(broadcastrecipient.FieldWorkspaceID), ws),
					sql.GTE(sentAt, since),
					sql.LT(sentAt, until),
				)).
				GroupBy("day")
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	type counts struct{ sent, opened, clicked int }
	byDay := make(map[string]counts, len(rows))
	for _, r := range rows {
		byDay[r.Day] = counts{sent: r.Sent, opened: r.Opened, clicked: r.Clicked}
	}

	var points []siteapi.SiteAnalyticsPoint
	for d := since; d.Before(until); d = d.AddDate(0, 0, 1) {
		day := d.Format(analyticsDayFormat)
		c := byDay[day]
		points = append(points, siteapi.SiteAnalyticsPoint{
			Date:    day,
			Sent:    int32(c.sent),
			Opened:  int32(c.opened),
			Clicked: int32(c.clicked),
		})
	}
	return points, nil
}

// ratio is num/denom as a float32 in [0,1], guarding against a zero denominator.
// Mirrors the per-broadcast rate semantics in the resources package.
func ratio(num, denom int) float32 {
	if denom <= 0 {
		return 0
	}
	return float32(num) / float32(denom)
}
