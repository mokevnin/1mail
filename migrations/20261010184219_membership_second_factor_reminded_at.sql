-- +goose Up
-- modify "memberships" table
ALTER TABLE "memberships" ADD COLUMN "second_factor_reminded_at" timestamptz NULL;

-- +goose Down
-- reverse: modify "memberships" table
ALTER TABLE "memberships" DROP COLUMN "second_factor_reminded_at";
