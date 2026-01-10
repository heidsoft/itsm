data "external_schema" "ent" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-ent",
    "load",
    "--path",
    "./ent/schema",
  ]
}

env "local" {
  src = data.external_schema.ent.url
  dev = getenv("DATABASE_DEV_URL")
  url = getenv("DATABASE_URL")
  migration {
    dir = "file://ent/migrate"
  }
}
