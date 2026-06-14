# ADR-005: sqlc for Type-Safe SQL over ORM

**Status:** Accepted
**Date:** 2026-01-16
**Decision Makers:** Project Owner

## Context

Every service needs a data access layer. The Go ecosystem offers several approaches:

1. **ORM (GORM, Ent)** — Object-relational mapping, writes SQL for you
2. **Query Builder (squirrel, goqu)** — Programmatic SQL construction
3. **Raw SQL with database/sql** — Hand-written queries, manual scanning
4. **sqlc** — Write SQL, generate type-safe Go code

## Decision

We use **sqlc** to generate our data access layer from hand-written SQL queries.

## How It Works

### 1. Write SQL queries with annotations

```sql
-- name: GetStudentByID :one
SELECT id, student_number, first_name, last_name, email, department, advisor_id
FROM students
WHERE id = $1;

-- name: ListStudentsByDepartment :many
SELECT id, student_number, first_name, last_name, email
FROM students
WHERE department = $1
ORDER BY last_name
LIMIT $2 OFFSET $3;

-- name: CreateStudent :one
INSERT INTO students (student_number, first_name, last_name, email, department)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
```

### 2. Run `sqlc generate`

sqlc reads the SQL files and migration schemas, then generates:

- `models.go` — Structs matching your table schemas
- `db.go` — Database interface
- `{query_file}.sql.go` — Type-safe Go functions for each query

### 3. Use generated code in repository layer

```go
func (r *StudentRepo) GetByID(ctx context.Context, id pgtype.UUID) (*db.Student, error) {
    return r.queries.GetStudentByID(ctx, id)
}
```

## Rationale

**Why sqlc over an ORM (GORM):**

- **SQL is the interface**: We write real SQL. No learning a DSL, no "how do I do X in GORM" — if you know PostgreSQL, you know how to write queries.
- **No runtime reflection**: GORM uses reflection to map structs to tables. sqlc generates plain Go code at build time — no magic, no runtime overhead.
- **Compile-time safety**: If a query references a column that doesn't exist, `sqlc generate` fails. GORM queries fail at runtime.
- **Full PostgreSQL power**: CTEs, window functions, JSONB operators, array types — sqlc passes through whatever valid SQL you write. ORMs often can't express advanced queries.
- **Predictable queries**: What you write is what runs. No N+1 problems, no unexpected JOINs, no lazy loading surprises.

**Why sqlc over raw database/sql:**

- **No manual scanning**: With `database/sql`, you write `rows.Scan(&id, &name, &email, ...)` for every query. Miss a column? Runtime panic. Reorder columns? Bug. sqlc eliminates this entirely.
- **Parameter type safety**: sqlc generates functions with typed parameters. You can't pass a string where an int is expected.

**Trade-offs accepted:**

- **Code generation step**: Developers must run `make sqlc` after changing queries. Forgetting this leads to stale generated code.
- **Generated files must not be edited**: The `internal/db/` directory is regenerated entirely. Custom logic goes in the repository layer above it.
- **pgtype complexity**: sqlc with pgx v5 uses `pgtype` types (pgtype.Text, pgtype.UUID, etc.) instead of plain Go types. This requires conversion helpers (see `shared/utils/pgtype_helpers.go`).

## Consequences

- Each service has a `sql/queries/` directory with annotated SQL files.
- Each service has a `sql/migrations/` directory with Goose migration files.
- Generated code lives in `internal/db/` — **never edit these files**.
- The repository layer wraps generated code and handles pgtype conversions.
- `sqlc.yaml` in each service configures code generation settings.
- All developers must have sqlc installed (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`).
