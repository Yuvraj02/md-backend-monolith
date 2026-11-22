env "local" {
  url = getenv("DATABASE_URL")
  dev = getenv("DEV_DATABASE_URL")

  migration {
    dir = "file://migrations"
    # Neon-friendly: keep revision ledger in public instead of a separate schema.
    revisions_schema = "public"
  }
}

lint {
  destructive {
    error = true
  }
}
