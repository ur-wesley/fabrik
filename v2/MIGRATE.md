# SQL migrations

Forward-only numbered `.sql` files under `cli/internal/sqlmigrate/sql/{dolt,memory,graph}/`.

## schema_migrations

Every store records applied files:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  filename TEXT NOT NULL,
  applied_at TEXT NOT NULL
);
```

`version` is the numeric prefix of the filename (`001_foo.sql` → 1).

## Rules

1. Apply pending files in ascending `version` order. Skip already applied.
2. One transaction per file. On failure, roll back that file only.
3. If the database max version is greater than the binary’s max embedded version, exit `fabrik: store schema N is newer than this fabrik`. Never auto-downgrade.
4. **SQLite** (`memory.db`, `graph.db`): copy the db file to `*.bak` before the first pending migration in a run. Restore `*.bak` if any pending migration fails.
5. **Dolt**: after a successful migration file, `CALL DOLT_COMMIT('-m', 'schema migration <filename>')`.

## Source of truth

- `001_*.sql` for dolt is generated from [SCHEMA.sql](SCHEMA.sql) plus `meta` and `schema_migrations` bootstrap.
- `001_*.sql` for memory from [MEMORY.sql](MEMORY.sql) plus `schema_migrations`.
- `001_*.sql` for graph from [GRAPH.sql](GRAPH.sql) plus `schema_migrations`.

Do not hand-edit production databases. Add `002_*.sql` for changes.

## Commands

`fabrik init` and `fabrik setup` run migrations on create. `fabrik check` prints `schema` per store.
