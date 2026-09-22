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
```

Every JSON response — success or error — follows the same envelope: `{ timestamp, status, items, error }`, produced by `internal/api/presenters`.

See [`CLAUDE.md`](CLAUDE.md) for the full architecture writeup, Fiber v2→v3 migration notes, and testing conventions.
