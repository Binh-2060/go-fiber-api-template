# go-fiber-api-template

A minimal boilerplate for a Go REST API built on [Fiber v3](https://github.com/gofiber/fiber). It's meant to be forked as the starting point for a real service: the layers under `internal/api/` are wired up but left as placeholders, and [`examples/crud`](examples/crud/README.md) shows the fully worked version of every one of them.

Request flow: `main.go` → global middleware → versioned route group → feature routes → controller → service → presenter.

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
internal/config/*        one package per global middleware (cors, helmet, logger, ...)
internal/api/             routes / controllers / services / repositories / models /
                          schemas / presenters / validators / middlewares
pkg/db                   Postgres pool (pgx/v5) + transaction helpers
pkg/jwt                  JWT sign/verify (HS256 Manager, RS256 RSAManager, both TokenManager), not yet wired to a route
examples/api              smallest possible feature skeleton (route → handler → envelope)
examples/crud             full worked CRUD feature (users), compiled but not mounted
migrations/               SQL migration files
```

Every JSON response — success or error — follows the same envelope: `{ timestamp, status, items, error }`, produced by `internal/api/presenters`.

See [`CLAUDE.md`](CLAUDE.md) for the full architecture writeup, Fiber v2→v3 migration notes, and testing conventions.

## Building features with the Claude Code skills

This repo ships two Claude Code skills under `.claude/skills/` that scaffold new features by imitating the `examples/` folder file-for-file, so everything a new feature adds matches the existing structure, naming and style.

### `/new-api` — feature skeleton, no database

Use this when you want an empty starting point for a feature that isn't a straightforward CRUD table — a route group, one handler, and empty placeholder packages for the layers you'll fill in yourself.

```
/new-api orders
```

Produces a mounted route, one handler returning the response envelope, and empty `services/`, `repositories/`, `schemas/requestbody/`, `schemas/responsebody/` packages — no database, no migration, no tests, nothing filled in for you.

### `/new-feature` — full CRUD from a migration

Use this when you have a table in `migrations/` and want the complete CRUD stack for it: models, repository, service, controller, routes, request/response schemas, and (optionally) tests — all mirroring `examples/crud`.

```
/new-feature "products.sql"
```

The skill reads only the named migration file (it won't browse `migrations/` for you), derives the feature name and route path from the table, and asks once whether to generate tests. It never adds anything `examples/crud` doesn't already have — no extra endpoints, filters, or dependencies — and never edits shared code (`internal/api/presenters`, `internal/api/validators`, `pkg/db`, `internal/config/*`), only mounting the new routes in `internal/api/routes/routes.go`.

### Choosing between them

| | `/new-api` | `/new-feature` |
| --- | --- | --- |
| Input | a feature name | a migration filename in `migrations/` |
| Output | route + handler skeleton, empty layers | full CRUD: model, repository, service, controller, schemas |
| Database | none | yes, backed by the named migration |
| Tests | none | optional, on request |

If you ask for create/list/update/delete on a specific table, `/new-feature` is the right one; if you're scaffolding something else entirely and want to write the logic yourself, use `/new-api`.
