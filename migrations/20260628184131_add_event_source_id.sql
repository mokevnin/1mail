-- Modify "events" table
ALTER TABLE "public"."events" ADD COLUMN "source_id" character varying NULL;
-- Create index "events_source_id_key" to table: "events"
CREATE UNIQUE INDEX "events_source_id_key" ON "public"."events" ("source_id");
