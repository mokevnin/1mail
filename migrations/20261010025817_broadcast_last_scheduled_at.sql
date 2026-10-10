-- +goose Up
-- modify "broadcasts" table
ALTER TABLE "broadcasts" ADD COLUMN "last_scheduled_at" timestamptz NULL;

-- +goose Down
-- reverse: modify "broadcasts" table
ALTER TABLE "broadcasts" DROP COLUMN "last_scheduled_at";
