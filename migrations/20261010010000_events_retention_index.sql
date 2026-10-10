-- +goose Up
-- Create index "events_created_at_analytical_idx" to table: "events"
-- Backs the Event retention delete (ADR 0019); evidentiary Events are excluded.
CREATE INDEX "events_created_at_analytical_idx" ON "public"."events" ("created_at") WHERE action NOT IN ('marketing.confirmed', 'email.complained', 'email.unsubscribed') AND NOT (action = 'email.bounced' AND COALESCE(properties->>'bounceKind', '') = 'permanent');

-- +goose Down
-- Reverse: create index "events_created_at_analytical_idx" to table: "events"
DROP INDEX "public"."events_created_at_analytical_idx";
