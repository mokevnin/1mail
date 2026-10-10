-- +goose Up
-- modify "workspaces" table
ALTER TABLE "workspaces" ADD COLUMN "retention_days" bigint NULL;

-- +goose Down
-- reverse: modify "workspaces" table
ALTER TABLE "workspaces" DROP COLUMN "retention_days";
