env "local" {
  src = "ent://ent/schema"
  dev = "docker://postgres/16/dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
