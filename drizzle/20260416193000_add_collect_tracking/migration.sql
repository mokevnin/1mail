CREATE TABLE "tracking_profiles" (
  "id" bigserial PRIMARY KEY NOT NULL,
  "subject_id" text NOT NULL,
  "email" text,
  "phone" text,
  "traits" jsonb DEFAULT '{}'::jsonb NOT NULL,
  "created_at" timestamp with time zone DEFAULT now() NOT NULL,
  "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
  CONSTRAINT "tracking_profiles_subject_id_unique" UNIQUE("subject_id"),
  CONSTRAINT "tracking_profiles_email_unique" UNIQUE("email"),
  CONSTRAINT "tracking_profiles_phone_unique" UNIQUE("phone")
);
--> statement-breakpoint
CREATE TABLE "tracking_visitors" (
  "id" bigserial PRIMARY KEY NOT NULL,
  "visitor_id" text NOT NULL,
  "profile_id" bigint,
  "created_at" timestamp with time zone DEFAULT now() NOT NULL,
  "updated_at" timestamp with time zone DEFAULT now() NOT NULL,
  "last_seen_at" timestamp with time zone DEFAULT now() NOT NULL,
  CONSTRAINT "tracking_visitors_visitor_id_unique" UNIQUE("visitor_id")
);
--> statement-breakpoint
ALTER TABLE "tracking_visitors"
ADD CONSTRAINT "tracking_visitors_profile_id_tracking_profiles_id_fk"
FOREIGN KEY ("profile_id") REFERENCES "public"."tracking_profiles"("id")
ON DELETE set null ON UPDATE no action;
