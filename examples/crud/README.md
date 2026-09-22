# Example: users CRUD

A complete feature built on a `users` table, showing how a request travels through every layer of this template.

```
routes → controllers → services → repositories → pkg/db
                ↕                       ↕
             schemas                  models

exceptions ← returned by repositories, sent as err.Error() by controllers
```

## Building your own CRUD feature

Follow [`.ai/workings/00-crud.md`](../../.ai/workings/00-crud.md): it starts from your table's generated reference (`internal/api/models/gen/<table>.gen.go`), not from this example's `models/user.go`, and derives the repository model and request validators from it.

## Endpoints

| Method   | Path                | Body / query                    | Success |
| -------- | ------------------- | ------------------------------- | ------- |
| `POST`   | `/users/newData`    | `{name}`                        | 200     |
| `GET`    | `/users/getData`    | `?page=1&per_page=20&q=ada`     | 200     |
| `GET`    | `/users/info/:id`   | —                               | 200     |
| `PUT`    | `/users/update/:id` | `{name}`                        | 200     |
| `DELETE` | `/users/:id`        | —                               | 200     |

Errors: a malformed id or bad body → 400; any other error → 500 with its message in `error` (e.g. `"user not found"`).

Every response uses the standard envelope from `internal/api/presenters` — `{timestamp, status, items, error}` — and `GET /users/getData` adds the pagination block via `ResponseSuccessListData`.

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

The example reads `id` and `name`, typed from `internal/api/models/gen/users.gen.go`, so it runs against the `users` table the generator read:

```bash
curl -X POST localhost:$PORT/api/$API_VERSION/users/newData \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada"}'
```

## What each layer is doing

**`models/`** — the table, as a Go struct. Field order matches the repository's column list so one scan order serves every query.

**`repositories/`** — SQL only. Resolves its connection with `db.Q(ctx)`, never `db.Pool()`, so the exact same function runs inside or outside a transaction. Returns `exceptions.ErrUserNotFound` instead of an HTTP status — nothing at this level knows the caller is a web server. `DeleteUser` checks rows affected, so deleting a missing id is not-found too.

**`exceptions/`** — every error the feature returns on purpose (`ErrUserNotFound`, `ErrNoUpdateFields`). It imports nothing from the feature, so any layer can use these values without an import cycle. They are plain Go error values, checked with `errors.Is`.

**`services/`** — orchestration. `CreateUser`, `UpdateUser` and `DeleteUser` write inside `db.ExecTx`, so an error rolls the write back. `ListUsers` runs its count and its page as two separate queries, sharing one filter: if rows are added or deleted between them, `total_page` can be off by a little. That is normal for a paginated list and keeps the code simple; wrap both in `db.ExecTxOptions` with `pgx.RepeatableRead` only if a feature needs them to agree exactly (a plain `ExecTx` is not enough — Postgres's default isolation can still see changes between the two queries).

**`schemas/`** — the wire contract, split from the model on purpose. Request bodies are derived from `internal/api/models/gen/users.gen.go` (see `.ai/workings/00-crud.md`): `name` is `not null varchar(255)`, so it is `required,max=255`; `id` is filled by the database and never sent. The repository's `UpdateUser` takes `*string` per column and skips nils in the `SET` clause, so a partial update is ready once the table has more columns.

**`controllers/`** — thin. Validate, call a service, shape the output. A service error is returned as `fiber.NewError(500, err.Error())`, so the client reads the exception's message. Errors are returned, not rendered: `main.go`'s `ErrorHandler` is the one place an error becomes JSON.

## Raw SQL vs. query builder

The repository uses both, on purpose — the split is the lesson.

**Raw SQL** for `CreateUser`, `GetUserByID`, `DeleteUser`. The statement is fixed and known at compile time, and `INSERT INTO users (name) VALUES ($1) RETURNING …` is already exactly what it says. A builder would make it longer and add an `err` return for a failure that can only be a programming mistake.

**[Squirrel](https://github.com/Masterminds/squirrel)** for `ListUsers`, `CountUsers` and `UpdateUser`, where the SQL genuinely is not known until runtime:

- `?q=` adds a `WHERE`, `UpdateUser` sets only the columns that are non-nil. Doing that by hand means concatenating clause fragments and hand-numbering `$1, $2, $3` as conditions come and go — the source of both injection bugs and placeholder off-by-ones.
- `applyUserFilter` is shared by the list and the count, so a filtered page and its `total_page` can't be computed from different `WHERE` clauses.

Two things to know about using Squirrel here:

```go
// Squirrel defaults to '?' placeholders (MySQL). Postgres needs $1.
var builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
```

and it is used **only as a string builder**. `RunWith`, `.Query`, `.Scan` are all built on `database/sql`, and this project talks to Postgres through native pgx — so every call ends in `.ToSql()` and hands the result to `db.Q(ctx)`.

Values always stay arguments, never SQL text. `sq.ILike{"name": "%ada%"}` renders as `name ILIKE $1`, so a filter of `' OR '1'='1` matches nothing instead of being clever.

## Tests

```bash
go test -tags=integration ./examples/crud/...  # needs Postgres and .env
```

### Start with controller tests

For now we test at the **controller layer**: `tests/user_route_test.go` sends real HTTP requests through the Fiber app (`app.Test`, no open port) and checks what a client sees:

- status code and the `{timestamp, status, items, error}` envelope
- validation: a bad body or a malformed id answers 400
- errors: an id with no row answers 500 with `"user not found"` on get, update and delete
- the data: create, then read it back; update, then read it back; delete, then it's gone
- lists: `[]` instead of `null`, and the pagination block

One request runs the whole stack — route → controller → service → repository → Postgres — so these few tests already cover every layer, and they're the easiest to read. For a real feature, write `internal/api/tests/<feature>_route_test.go` first, copying this file's `newTestApp` / `do` helpers and `main_test.go`:

```go
func TestHTTP_CreateThenGet(t *testing.T) {
	app := newTestApp()
	m := marker(t)

	status, env := do(t, app, http.MethodPost, "/users/newData", map[string]string{"name": m + "Ada"})
	if status != http.StatusOK {
		t.Fatalf("POST /users/newData = %d, want 200 (%v)", status, env.Error)
	}
	// ...then GET /users/info/:id and compare
}
```

**Isolation is by marker, not by truncation.** `marker(t)` wraps `testsupport.Marker(t, "users", "name")` from `internal/testsupport`: it returns a random prefix, every row a test creates carries it in `name`, lists filter on it, and cleanup deletes only rows that have it. For another table, pass its own table and a text column. So tests never touch data they didn't create and can share one database. The prefix has no `_` or `%`, because both are `LIKE` wildcards.

Repository and service tests are left out on purpose, to keep the example simple; add them later when a feature needs checks a controller test can't see.

## Fiber v3 notes

Points where v3 differs from the v2 idioms in most online examples:

- Handlers take `fiber.Ctx` **by value** — `Ctx` is an interface in v3.
- `c.Context()` returns a `context.Context` (use `c.RequestCtx()` for the fasthttp one).
- Binding replaces parsing: `validators.ParseAndValidateBody` / `ParseAndValidateQueryParam` wrap `c.Bind().Body` / `.Query`. Query tags are `query:"..."`; typed helpers like `QueryInt` are gone.
- Path params could also be bound with `c.Bind().URI(&out)` and `uri:"id"` tags (v2 used `params:"..."`). This example uses `c.Params` + `validators.ValidateUuid`, which is less ceremony for a single param.
