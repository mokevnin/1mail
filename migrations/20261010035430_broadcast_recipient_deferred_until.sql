-- +goose Up
-- modify "broadcast_recipients" table
ALTER TABLE "broadcast_recipients" ADD COLUMN "deferred_until" timestamptz NULL;

-- +goose Down
-- reverse: modify "broadcast_recipients" table
ALTER TABLE "broadcast_recipients" DROP COLUMN "deferred_until";
