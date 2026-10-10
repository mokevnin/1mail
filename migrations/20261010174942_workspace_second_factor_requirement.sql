-- +goose Up
-- modify "workspaces" table
ALTER TABLE "workspaces" ADD COLUMN "second_factor_required_at" timestamptz NULL;

-- +goose Down
-- reverse: modify "workspaces" table
ALTER TABLE "workspaces" DROP COLUMN "second_factor_required_at";
