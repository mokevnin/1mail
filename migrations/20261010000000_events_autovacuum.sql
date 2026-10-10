-- +goose Up
-- The events table is append-heavy and pruned in batches (ADR 0019), so vacuum and
-- analyze it at a lower dead/changed-tuple ratio than the 20% / 10% defaults.
-- The outbox tables get theirs in events.InitSchema: they are created at boot,
-- not by migrations.
ALTER TABLE "events" SET (autovacuum_vacuum_scale_factor = 0.02, autovacuum_vacuum_threshold = 1000, autovacuum_analyze_scale_factor = 0.02);

-- +goose Down
ALTER TABLE "events" RESET (autovacuum_vacuum_scale_factor, autovacuum_vacuum_threshold, autovacuum_analyze_scale_factor);
