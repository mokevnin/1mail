-- Modify "users" table
ALTER TABLE "public"."users" ADD COLUMN "updated_at" timestamptz NOT NULL;
