-- Drop "transactional_emails" table (ADR 0015: replaced by outbound_messages; the app is pre-release, no data is carried over)
DROP TABLE "public"."transactional_emails";
