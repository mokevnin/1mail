-- Modify "workspaces" table
ALTER TABLE "public"."workspaces" ADD COLUMN "collect_key" character varying NOT NULL;
-- Create index "workspaces_collect_key_key" to table: "workspaces"
CREATE UNIQUE INDEX "workspaces_collect_key_key" ON "public"."workspaces" ("collect_key");
-- Modify "api_tokens" table
ALTER TABLE "public"."api_tokens" DROP CONSTRAINT "api_tokens_workspaces_api_tokens", ALTER COLUMN "workspace_id" SET NOT NULL, ADD CONSTRAINT "api_tokens_workspaces_api_tokens" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "contacts" table
ALTER TABLE "public"."contacts" DROP CONSTRAINT "contacts_workspaces_contacts", ALTER COLUMN "workspace_id" SET NOT NULL, ADD CONSTRAINT "contacts_workspaces_contacts" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "events" table
ALTER TABLE "public"."events" DROP CONSTRAINT "events_workspaces_events", ALTER COLUMN "workspace_id" SET NOT NULL, ADD CONSTRAINT "events_workspaces_events" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "tracking_profiles" table
ALTER TABLE "public"."tracking_profiles" DROP CONSTRAINT "tracking_profiles_workspaces_tracking_profiles", ALTER COLUMN "workspace_id" SET NOT NULL, ADD CONSTRAINT "tracking_profiles_workspaces_tracking_profiles" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Drop index "tracking_visitors_visitor_id_key" from table: "tracking_visitors"
DROP INDEX "public"."tracking_visitors_visitor_id_key";
-- Modify "tracking_visitors" table
ALTER TABLE "public"."tracking_visitors" ADD COLUMN "workspace_id" bigint NOT NULL, ADD CONSTRAINT "tracking_visitors_workspaces_tracking_visitors" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "tracking_visitors_visitor_id_workspace_id" to table: "tracking_visitors"
CREATE UNIQUE INDEX "tracking_visitors_visitor_id_workspace_id" ON "public"."tracking_visitors" ("visitor_id", "workspace_id");
