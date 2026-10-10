-- +goose Up
-- modify "integrations" table
ALTER TABLE "integrations" ADD COLUMN "provider_max_per_second" bigint NULL, ADD COLUMN "provider_max_per_day" bigint NULL, ADD COLUMN "provider_quota_checked_at" timestamptz NULL, ADD COLUMN "provider_quota_unavailable" boolean NOT NULL DEFAULT false;

-- +goose Down
-- reverse: modify "integrations" table
ALTER TABLE "integrations" DROP COLUMN "provider_quota_unavailable", DROP COLUMN "provider_quota_checked_at", DROP COLUMN "provider_max_per_day", DROP COLUMN "provider_max_per_second";
