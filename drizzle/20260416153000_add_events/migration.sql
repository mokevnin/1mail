CREATE TABLE "events" (
  "id" bigserial PRIMARY KEY NOT NULL,
  "subject_id" text NOT NULL,
  "email" text,
  "phone" text,
  "action" text NOT NULL,
  "properties" jsonb,
  "occurred_at" timestamp with time zone,
  "prospect" boolean,
  "created_at" timestamp with time zone DEFAULT now() NOT NULL
);
