-- +goose Up
-- Modify "api_tokens" table
ALTER TABLE "api_tokens" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "automation_runs" table
ALTER TABLE "automation_runs" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "automations" table
ALTER TABLE "automations" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "broadcast_recipients" table
ALTER TABLE "broadcast_recipients" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "broadcasts" table
ALTER TABLE "broadcasts" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "confirmations" table
ALTER TABLE "confirmations" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "contacts" table
ALTER TABLE "contacts" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "custom_fields" table
ALTER TABLE "custom_fields" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "email_templates" table
ALTER TABLE "email_templates" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "events" table
ALTER TABLE "events" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ADD COLUMN "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- Modify "integrations" table
ALTER TABLE "integrations" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "invitations" table
ALTER TABLE "invitations" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "memberships" table
ALTER TABLE "memberships" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "oauth_clients" table
ALTER TABLE "oauth_clients" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ADD COLUMN "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- Modify "oauth_codes" table
ALTER TABLE "oauth_codes" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ADD COLUMN "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- Modify "outbound_messages" table
ALTER TABLE "outbound_messages" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "segments" table
ALTER TABLE "segments" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "sending_domains" table
ALTER TABLE "sending_domains" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "suppressions" table
ALTER TABLE "suppressions" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "tags" table
ALTER TABLE "tags" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ADD COLUMN "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- Modify "unsubscribes" table
ALTER TABLE "unsubscribes" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "users" table
ALTER TABLE "users" ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "visitors" table
ALTER TABLE "visitors" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "webhook_endpoints" table
ALTER TABLE "webhook_endpoints" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
-- Modify "workspaces" table
ALTER TABLE "workspaces" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;

-- +goose Down
