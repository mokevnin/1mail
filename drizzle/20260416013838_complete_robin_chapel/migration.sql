CREATE TABLE "contacts" (
	"id" serial PRIMARY KEY,
	"email" text NOT NULL UNIQUE,
	"first_name" text,
	"last_name" text,
	"status" text DEFAULT 'active' NOT NULL,
	"time_zone" text,
	"custom_fields" jsonb,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "users" (
	"id" serial PRIMARY KEY,
	"name" text NOT NULL,
	"email" text NOT NULL UNIQUE,
	"created_at" timestamp DEFAULT now() NOT NULL
);
