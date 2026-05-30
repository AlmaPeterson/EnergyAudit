This folder contains SQL migration files for the file-upload server.

To apply migrations manually using `psql`:

```bash
# example using environment variables
PGHOST=localhost PGUSER=postgres PGPASSWORD=secret PGDATABASE=mydb psql -f migrations/001_create_tables.sql
```

The project also runs schema creation automatically in `repositories/fileupload/postgres_file_upload_repository.go` when the repository initializes. Use these SQL files for migration systems or manual application.
