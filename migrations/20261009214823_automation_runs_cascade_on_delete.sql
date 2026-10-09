-- +goose Up
-- modify "automation_runs" table
ALTER TABLE "automation_runs" DROP CONSTRAINT "automation_runs_automations_runs", ADD CONSTRAINT "automation_runs_automations_runs" FOREIGN KEY ("automation_id") REFERENCES "automations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;

-- +goose Down
-- reverse: modify "automation_runs" table
ALTER TABLE "automation_runs" DROP CONSTRAINT "automation_runs_automations_runs", ADD CONSTRAINT "automation_runs_automations_runs" FOREIGN KEY ("automation_id") REFERENCES "automations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
