-- +goose Up
-- modify "users" table
ALTER TABLE "users" ADD COLUMN "session_epoch" bigint NOT NULL DEFAULT 0;

-- +goose Down
-- reverse: modify "users" table
ALTER TABLE "users" DROP COLUMN "session_epoch";
