package events

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// lagQuery reports, per registered consumer group, the age in seconds of the
// oldest outbox row past that group's cursor (0 when the group is caught up).
// The cursor is the (last_processed_transaction_id, offset_acked) tuple the
// watermill offsets adapter keeps; a group with no offsets row yet has not
// consumed anything, so its cursor is the start of the table. The outbox has no
// published flag, so "past the cursor" is the only definition of pending.
const lagQuery = `
SELECT g.consumer_group,
       COALESCE(EXTRACT(EPOCH FROM (LOCALTIMESTAMP - MIN(e.created_at))), 0)::float8
FROM unnest($1::text[]) AS g(consumer_group)
LEFT JOIN watermill_offsets_domain_events o ON o.consumer_group = g.consumer_group
LEFT JOIN watermill_domain_events e
       ON (e.transaction_id, e."offset")
        > (COALESCE(o.last_processed_transaction_id, '0'::xid8), COALESCE(o.offset_acked, 0))
GROUP BY g.consumer_group`

// RegisterLagGauge exports outbox.lag (seconds, one sample per consumer group in
// ConsumerGroups) on the global meter provider. The label is the group name only:
// a bounded technical dimension. The query runs once per scrape. The returned
// function unregisters the callback.
func RegisterLagGauge(db *sql.DB) (func() error, error) {
	meter := otel.Meter(otelScope)
	gauge, err := meter.Float64ObservableGauge(
		"outbox.lag",
		metric.WithUnit("s"),
		metric.WithDescription("Age of the oldest domain-event outbox row past each consumer group's cursor."),
	)
	if err != nil {
		return nil, fmt.Errorf("outbox lag gauge: %w", err)
	}
	reg, err := meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		rows, err := db.QueryContext(ctx, lagQuery, ConsumerGroups)
		if err != nil {
			return fmt.Errorf("outbox lag: %w", err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var group string
			var lag float64
			if err := rows.Scan(&group, &lag); err != nil {
				return fmt.Errorf("outbox lag scan: %w", err)
			}
			o.ObserveFloat64(gauge, lag, metric.WithAttributes(attribute.String("consumer_group", group)))
		}
		return rows.Err()
	}, gauge)
	if err != nil {
		return nil, fmt.Errorf("outbox lag callback: %w", err)
	}
	return reg.Unregister, nil
}
