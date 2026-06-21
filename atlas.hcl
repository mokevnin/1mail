// https://atlasgo.io/faq/dotenv-files

env "local" {
  src = "ent://ent/schema"
  // Target DB and the scratch "dev" DB atlas uses to compute diffs both come from
  // the environment (set by the compose `backend` service), so atlas runs inside a
  // container without needing the docker daemon (no `docker://…` dev URL).
  url = getenv("DATABASE_URL")
  dev = getenv("ATLAS_DEV_URL")
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
