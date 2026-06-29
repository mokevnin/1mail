-- Create index "event_workspace_id_email_action" to table: "events"
CREATE INDEX "event_workspace_id_email_action" ON "public"."events" ("workspace_id", "email", "action");
