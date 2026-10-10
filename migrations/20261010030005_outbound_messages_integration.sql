-- +goose Up
-- modify "outbound_messages" table
ALTER TABLE "outbound_messages" ADD COLUMN "integration_id" bigint NULL;
-- create index "outboundmessage_workspace_id_integration_id_sent_at" to table: "outbound_messages"
CREATE INDEX "outboundmessage_workspace_id_integration_id_sent_at" ON "outbound_messages" ("workspace_id", "integration_id", "sent_at");

-- +goose Down
-- reverse: create index "outboundmessage_workspace_id_integration_id_sent_at" to table: "outbound_messages"
DROP INDEX "outboundmessage_workspace_id_integration_id_sent_at";
-- reverse: modify "outbound_messages" table
ALTER TABLE "outbound_messages" DROP COLUMN "integration_id";
