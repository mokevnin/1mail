// https://atlasgo.io/faq/dotenv-files

env "local" {
  // No `src`: the community Atlas binary cannot read ent://. Migrations are written by
  // `mise run db:generate` (cmd/db generate, ent's own diff engine); this env only
  // applies them (`migrate apply` reads url + migration.dir).
  // Target DB and the scratch "dev" DB atlas uses to compute diffs both come from
  // the environment (set by the compose `backend` service), so atlas runs inside a
  // container without needing the docker daemon (no `docker://…` dev URL).
  url = getenv("DATABASE_URL")
  dev = getenv("ATLAS_DEV_URL")
  migration {
    dir = "file://migrations?format=goose"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
