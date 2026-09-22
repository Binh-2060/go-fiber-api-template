# go-fiber-api-template

A minimal boilerplate for a Go REST API built on [Fiber v3](https://github.com/gofiber/fiber). It's meant to be forked as the starting point for a real service: the layers under `internal/api/` are wired up but left as placeholders, and [`examples/crud`](examples/crud/README.md) shows the fully worked version of every one of them.

Request flow: `main.go` → global middleware → versioned route group → feature routes → controller → service → repository → presenter.

## Requirements

- Go 1.25+
- PostgreSQL (only needed for routes that touch the database)

## Getting started

```bash
cp .env.development .env   # loaded via godotenv.Load(), only .env is read at runtime
go run cmd/api/main.go
```

`.env.development` is the tracked template; `.env` is gitignored and is what actually gets loaded. Required vars: `GO_ENV`, `API_NAME`, `API_VERSION`, `PORT`. Database vars (`DATABASE_URL` or the `DB_*` fields) are only needed once a feature talks to Postgres.

```bash
go test ./...                    # pure logic, no database
go test -tags=integration ./...  # everything else, needs Postgres and the vars in .env
```

## Layout

```
cmd/api/main.go          composition root — middleware, routing, graceful shutdown
cmd/gorm/main.go         dev tool: generates reference structs into internal/api/models/gen (never imported)
internal/config/*        one package per global middleware (cors, helmet, logger, ...)
internal/api/             routes / controllers / services / repositories / models /
                          schemas / presenters / validators / middlewares
internal/testsupport     shared integration-test setup (TestMain, row Marker); feature tests go in internal/api/tests
pkg/db                   Postgres pool (pgx/v5) + transaction helpers
pkg/jwt                  JWT sign/verify (HS256 Manager, RS256 RSAManager, both TokenManager), not yet wired to a route
examples/api              smallest possible feature skeleton (route → handler → envelope)
examples/crud             full worked CRUD feature (users), compiled but not mounted
examples/login            JWT login (RS256) with in-memory users, compiled but not mounted
```

Every JSON response — success or error — follows the same envelope: `{ timestamp, status, items, error }`, produced by `internal/api/presenters`.

## Model types (GORM gen)

```bash
go run ./cmd/gorm   # from the repo root; same DATABASE_URL / DB_* vars as the API
```

Writes one struct per table to `internal/api/models/gen/<table>.gen.go`.

- **Reference only — never imported.** The `.gen.go` struct shows each column's Go type, SQL type (`type:` tag) and nullability. Use it to check your own models, scans and validators; never import `internal/api/models/gen` or `gorm.io/*` outside `cmd/gorm`.
- **Repositories use their own models** in `internal/api/models/`, written by hand with only the columns the query returns — a table row (`User`), a join, or a custom shape like a report row (`SalesReportRow`). All queries are hand-written on `pkg/db` (`db.Q(ctx)`).
- **Match the reference types** so `Scan` can't fail: same Go type per column, pointer where the column is nullable (a non-pointer field fails on `NULL`). Aggregates and `LEFT JOIN` columns have no `.gen.go` line — `count(*)` is `int64`, and anything that can be `NULL` is a pointer.
- **Nullable columns are pointers** (`*string`, `*int64`, `*time.Time`); a column is nullable when its `gorm` tag has no `not null`. pgx scans `NULL` into `nil` — no `COALESCE` needed.
- **Send `null` to the client, not a zero value.** Keep the pointer in the response struct so `nil` encodes as `null`; never dereference it to `""` / `0`, which a client can't tell from a real value. Check for `nil` before using it in Go.

  ```go
  type User struct {
      ID       string  `json:"id"`
      Nickname *string `json:"nickname"` // null when the column is NULL
  }
  ```
- **Lists are `[]`, never `null`.** Build response slices with `make([]T, 0, n)`; a `nil` slice encodes as `null`.
- **Prefer `NOT NULL DEFAULT` in the schema** when empty and "none" mean the same thing (e.g. `nickname DEFAULT ''`) — it generates a plain type and needs no nil checks. Keep a column nullable only when `NULL` has its own meaning (`deleted_at`, optional foreign key).
- **Don't edit `.gen.go` files.** Re-run the generator after a schema change.

See [`AGENT.md`](AGENT.md) for the full architecture writeup, Fiber v2→v3 migration notes, and testing conventions.
