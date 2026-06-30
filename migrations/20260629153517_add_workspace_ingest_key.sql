-- Modify "workspaces" table: add the inbound-webhook routing key. Added nullable
-- first so existing workspaces can be backfilled with unique generated keys
-- before the NOT NULL + uniqueness constraints are enforced.
ALTER TABLE "public"."workspaces" ADD COLUMN "ingest_key" character varying;
UPDATE "public"."workspaces"
  SET "ingest_key" = 'omik_' || replace(gen_random_uuid()::text, '-', '')
  WHERE "ingest_key" IS NULL;
ALTER TABLE "public"."workspaces" ALTER COLUMN "ingest_key" SET NOT NULL;
-- Create index "workspaces_ingest_key_key" to table: "workspaces"
CREATE UNIQUE INDEX "workspaces_ingest_key_key" ON "public"."workspaces" ("ingest_key");
