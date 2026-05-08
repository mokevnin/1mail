env "local" {
  src = "ent://ent/schema"
  dev = "docker://postgres/18/dev"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
