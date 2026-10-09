-- +goose Up
-- Modify "segments" table
ALTER TABLE "segments" DROP COLUMN "type";

-- +goose Down
