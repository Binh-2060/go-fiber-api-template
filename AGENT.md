**Read and follow these rules before starting any work:**

@.ai/rules/00-code-begin.md
@.ai/rules/01-code-rules.md

**When unsure, ask — don't guess.** If a requirement is unclear, the docs and the code disagree, a rule conflicts with an example, or a file/table/var mentioned here doesn't exist, stop and ask the maintainer before writing code.

## What this is

Boilerplate Go REST API on [Fiber v3](https://github.com/gofiber/fiber) (v3.4.0), meant to be forked. **Requires Go 1.25+** (Fiber v3's own floor).

`.env` is loaded (bare `godotenv.Load()`) only when `GO_ENV` is unset — **`.env`, not `.env.development`**, which is just the tracked template to copy. Required: `GO_ENV`, `API_NAME`, `API_VERSION`, `PORT`. Postgres: `DATABASE_URL`, or `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE` (+ optional `DB_MAX_CONNS` etc., see `pkg/db`). JWT: `JWT_ISSUER`, `JWT_TTL` (needs a unit: `24h`), plus `JWT_SECRET` for HS256 (`jwt.Manager`) or `JWT_RSA_PRIVATE_KEY_PATH` / `JWT_RSA_PUBLIC_KEY_PATH` for RS256 (`jwt.RSAManager`, used by `examples/login`; a verify-only side needs just the public key).

## Fiber v3 conventions (differ from v2 examples online)

- **Handlers take `fiber.Ctx` by value, not `*fiber.Ctx`** — it's an interface. Same for `ErrorHandler` and helpers.
- **Bind, don't parse:** `c.Bind().Body(out)` / `.Query(out)` / `.URI(out)` with `uri:"..."` tags. `QueryInt` & co. are gone — use `fiber.Query[T](c, key)`.
- **Logger uses `Stream`, not `Output`.**
- **Request ID is not in Locals:** `${requestid}` in log formats, `requestid.FromContext(c)` in code.
- `ListenTLS*` is gone — pass a `fiber.ListenConfig` to `app.Listen`.

## Architecture

Request flow: `main.go` → global middleware → `/api/{API_VERSION}` group → feature routes → controller → service → repository → presenter.

- **`cmd/api/main.go`** — composition root: middleware (CORS → request ID → validators → compress → helmet), `/` and `/healthz`, `routes.SetRoutes`, graceful shutdown. Its `ErrorHandler` is the only place errors become JSON.
- **`cmd/gorm`** — generates reference structs into `internal/api/models/gen/<table>.gen.go`. Never imported: repositories use their own hand-written models in `internal/api/models/` (see README).
- **`internal/config/*`** — one package per global middleware, each a `SetXMiddleware(app)` called from `main.go`.
- **`internal/api/`** — `routes/` (`SetRoutes` is empty; register each feature's `Set<Feature>Route` here), `presenters/`, `validators/`, `schemas/` (`requestbody/sample.go`, `responsebody/sample.go` placeholders), `models/` (hand-written repository models; `models/gen/` is generated reference). Feature errors go in `exceptions/<feature>.go` (create the folder with the first one). `controllers/`, `services/`, `repositories/` exist but are empty; `middlewares/auth.go` is a stub.
- **`internal/api/validators`** — call `ParseAndValidateBody` / `ParseAndValidateQueryParam` / `ValidateUuid`; Fiber's `StructValidator` isn't configured, so `validate:` tags only run through these.
- **`pkg/db`** (pgx/v5) — repositories use `db.Q(ctx)`, **never `db.Pool()`**: it returns the in-flight tx or the pool. `db.ExecTx(ctx, fn)` commits on nil / rolls back on error; nested calls become savepoints; `ExecTxOptions` for isolation.
- **`pkg/jwt`** — HS256 `Manager` and RS256 `RSAManager`, both `TokenManager`; depend on the interface. Errors are deliberately coarse (`ErrInvalidToken` / `ErrExpiredToken`). `pkg/bcrypt` is an empty stub.
- **Examples** (compiled, not mounted): `examples/api` (skeleton), `examples/crud` (full users CRUD + tests), `examples/login` (JWT login, in-memory users). `examples/crud/presenters` is dead code — import `internal/api/presenters`.

**Response envelope:** every response is `{ "timestamp", "status" (1/0), "items", "error" }`. Success goes through `presenters.ResponseSuccess` / `ResponseSuccessListData` (pass `-1` for pagination when unused); never hand-roll `fiber.Map`.

## Tests

```bash
go test ./...                    # pure logic, no database — must stay green with no DB
go test -tags=integration ./...  # needs Postgres and .env
```

Start with controller (HTTP) tests like `examples/crud/tests/user_route_test.go`. A real feature's tests go in `internal/api/tests/<feature>_route_test.go` (one shared `tests` package with one `main_test.go`, created with the first feature). Shared setup is `internal/testsupport`: `testsupport.Main(m)` in `TestMain`, and `testsupport.Marker(t, table, column)` to isolate rows (never truncate). Confirm each test can fail.

## Logging

Mount `SetLoggerMiddlewareJSON` **before** any route on the group — Fiber skips middleware for routes registered earlier.
