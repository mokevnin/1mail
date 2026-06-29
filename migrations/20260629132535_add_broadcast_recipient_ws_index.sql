-- Create index "broadcastrecipient_workspace_id_sent_at" to table: "broadcast_recipients"
CREATE INDEX "broadcastrecipient_workspace_id_sent_at" ON "public"."broadcast_recipients" ("workspace_id", "sent_at");
