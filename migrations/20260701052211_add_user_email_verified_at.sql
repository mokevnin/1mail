-- Modify "users" table
ALTER TABLE "public"."users" ADD COLUMN "email_verified_at" timestamptz NULL;
