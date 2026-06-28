-- Modify "broadcasts" table
ALTER TABLE "public"."broadcasts" DROP COLUMN "body_html", DROP COLUMN "body_format", ADD COLUMN "body" character varying NOT NULL DEFAULT '';
-- Modify "email_templates" table
ALTER TABLE "public"."email_templates" DROP COLUMN "body_html", DROP COLUMN "body_format", ADD COLUMN "body" character varying NOT NULL DEFAULT '';
