-- Modify "broadcast_recipients" table
ALTER TABLE "broadcast_recipients" ADD COLUMN "outbound_message_id" bigint NULL;
-- Modify "broadcasts" table
ALTER TABLE "broadcasts" ADD COLUMN "skipped_count" bigint NOT NULL DEFAULT 0, ADD COLUMN "hold_reason" character varying NULL;
