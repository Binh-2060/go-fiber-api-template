# Example: users CRUD

A complete feature built on a `users` table, showing how a request travels through every layer of this template.

```
routes → controllers → services → repositories → pkg/db
                ↕                       ↕
             schemas                  models
```

## The table

```sql
create table users
(
    id         uuid                     default gen_random_uuid() not null primary key,
    name       varchar(200)             default 'N/A'             not null,
    created_at timestamp with time zone default now()             not null,
    surename   varchar(200)                                       not null
);
```

`surename` is spelled that way in the table; the code follows the schema rather than diverging from it. Rename the column if you want `surname`.

## Endpoints

| Method   | Path              | Body / query                    | Success |
| -------- | ----------------- | ------------------------------- | ------- |
| `POST`   | `/users`          | `{name, surename}`              | 201     |
| `POST`   | `/users/bulk`     | `{users: [{name, surename}]}`   | 201     |
| `GET`    | `/users`          | `?page=1&per_page=20&q=ada`     | 200     |
| `GET`    | `/users/:id`      | —                               | 200     |
| `PATCH`  | `/users/:id`      | `{name, surename}`              | 200     |
| `DELETE` | `/users/:id`      | —                               | 200     |

Every response uses the standard envelope from `internal/api/presenters` — `{timestamp, status, items, error}` — and `GET /users` adds the pagination block via `ResponseSuccessListData`.

## Wiring it up

The example is compiled but not mounted. `internal/api/routes/routes.go` starts
with an empty `SetRoutes`; to run the example, fill it in:

```go
import crudroutes "github.com/Binh-2060/go-application-template/examples/crud/routes"

func SetRoutes(router fiber.Router) {
	userRoutes := router.Group("/users")
	crudroutes.SetUserRoute(userRoutes)
}
```

Then create the table (the `create table` statement above) and hit it:

```bash
curl -X POST localhost:$PORT/api/$API_VERSION/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada","surename":"Lovelace"}'
```

## What each layer is doing

**`models/`** — the table, as a Go struct. Field order matches the repository's column list so one scan order serves every query.

**`repositories/`** — SQL only. Resolves its connection with `db.Q(ctx)`, never `db.Pool()`, so the exact same function runs inside or outside a transaction. Returns the `ErrUserNotFound` sentinel instead of an HTTP status — nothing at this level knows the caller is a web server.

**`services/`** — orchestration. `ListUsers` runs its count and its page inside one `db.ExecTx` so the two cannot disagree; `CreateUsers` wraps the whole loop in a transaction, so one bad row rolls back all of them.

**`schemas/`** — the wire contract, split from the model on purpose. `requestbody.UpdateUser` takes both fields as plain, required strings: `PATCH` here means "replace", not "merge". The repository's `UpdateUser` still takes `*string` per column and skips nils in the `SET` clause — that partial-update capability is still there at the repository layer, the request schema just doesn't expose it.

**`controllers/`** — thin. Validate, call a service, shape the output. Errors are returned, not rendered: `main.go`'s `ErrorHandler` is the one place an error becomes JSON.

## Raw SQL vs. query builder

The repository uses both, on purpose — the split is the lesson.

**Raw SQL** for `CreateUser`, `GetUserByID`, `DeleteUser`. The statement is fixed and known at compile time, and `INSERT INTO users (name, surename) VALUES ($1, $2) RETURNING …` is already exactly what it says. A builder would make it longer and add an `err` return for a failure that can only be a programming mistake.

**[Squirrel](https://github.com/Masterminds/squirrel)** for `ListUsers`, `CountUsers` and `UpdateUser`, where the SQL genuinely is not known until runtime:

- `?name=` adds a `WHERE`, `PATCH` sets only the columns that were sent. Doing that by hand means concatenating clause fragments and hand-numbering `$1, $2, $3` as conditions come and go — the source of both injection bugs and placeholder off-by-ones.
- `applyUserFilter` is shared by the list and the count, so a filtered page and its `total_page` can't be computed from different `WHERE` clauses.

Two things to know about using Squirrel here:

```go
// Squirrel defaults to '?' placeholders (MySQL). Postgres needs $1.
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
```

and it is used **only as a string builder**. `RunWith`, `.Query`, `.Scan` are all built on `database/sql`, and this project talks to Postgres through native pgx — so every call ends in `.ToSql()` and hands the result to `db.Q(ctx)`.

Values always stay arguments, never SQL text. `sq.ILike{"name": "%ada%"}` renders as `name ILIKE $1`, so a filter of `' OR '1'='1` matches nothing instead of being clever.

## Tests

```bash
go test ./examples/crud/...                    # SQL generation only, no database
go test -tags=integration ./examples/crud/...  # everything, needs Postgres
```

The split is deliberate. `user_repository_sql_test.go` is untagged and asserts
properties of the *generated string* — `$1` placeholders rather than MySQL `?`,
the `id` tiebreaker in `ORDER BY`, and filter values arriving as bound arguments
instead of SQL text. Those need no database, so they run in CI on every commit.

Everything else is behind `//go:build integration`, because it asserts things
only a real server can answer: what pgx returns for a missing row, whether
`ILIKE` is genuinely case-insensitive, whether a rollback actually removed the
rows. A plain `go test ./...` stays green without a container.

All of it lives in `tests/`, one package, rather than beside each package it
exercises:

| File | Covers |
| --- | --- |
| `tests/user_repository_sql_test.go` | Generated SQL. No database, no build tag |
| `tests/user_repository_test.go` | Sentinels, partial update, ILIKE, paging |
| `tests/user_service_test.go` | Bulk-create rollback, pagination defaults |
| `tests/user_route_test.go` | Status codes, validation, response envelope |
| `tests/main_test.go` | `TestMain` and the marker helper |

Testing from outside means the tests only see what a real caller sees, which is
a genuine benefit — but it has a cost worth being deliberate about. Anything a
test touches has to be exported, so `repositories.Builder`, `ApplyUserFilter`
and `BuildListUsersQuery` are public *only* because this package asserts against
them. Unexported values cannot be reached at all: the pagination defaults are
written as literals in the test, with a comment naming the real definition.

The Go standard library puts test files beside the code for exactly that reason.
Both layouts are fine; pick one and keep it.

**Isolation is by marker, not by truncation.** `testsupport.Marker(t)` returns a
random prefix, every row a test creates carries it in `name`, every query filters
on it, and `t.Cleanup` deletes only rows that have it. So the tests never touch
data they did not create and can run against a shared database. The prefix
contains no `_` or `%` on purpose — both are `LIKE` wildcards, and a marker that
matched other rows would delete them.

Some assertions are only reachable at one layer. The bulk-create rollback lives
in the service test because the `max=200` validate tag rejects an over-long value
before it could ever fail mid-transaction over HTTP.

## Fiber v3 notes

Points where v3 differs from the v2 idioms in most online examples:

- Handlers take `fiber.Ctx` **by value** — `Ctx` is an interface in v3.
- `c.Context()` returns a `context.Context` (use `c.RequestCtx()` for the fasthttp one).
- Binding replaces parsing: `validators.ParseAndValidateBody` / `ParseAndValidateQueryParam` wrap `c.Bind().Body` / `.Query`. Query tags are `query:"..."`; typed helpers like `QueryInt` are gone.
- Path params could also be bound with `c.Bind().URI(&out)` and `uri:"id"` tags (v2 used `params:"..."`). This example uses `c.Params` + `validators.ValidateUuid`, which is less ceremony for a single param.
