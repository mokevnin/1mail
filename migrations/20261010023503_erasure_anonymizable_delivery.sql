-- +goose Up
-- modify "broadcast_recipients" table
ALTER TABLE "broadcast_recipients" ALTER COLUMN "contact_id" DROP NOT NULL;
-- modify "outbound_messages" table
ALTER TABLE "outbound_messages" ALTER COLUMN "destination" DROP NOT NULL;

-- +goose Down
-- reverse: modify "outbound_messages" table
ALTER TABLE "outbound_messages" ALTER COLUMN "destination" SET NOT NULL;
-- reverse: modify "broadcast_recipients" table
ALTER TABLE "broadcast_recipients" ALTER COLUMN "contact_id" SET NOT NULL;
