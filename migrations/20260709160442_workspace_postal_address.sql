-- Modify "workspaces" table
ALTER TABLE "public"."workspaces" ADD COLUMN "postal_address" character varying NULL DEFAULT '';
