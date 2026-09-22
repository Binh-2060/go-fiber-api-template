# Code rules

Before implementing any task, read the example it matches and copy its structure — same folders, file names, naming and style:

- **Any feature** → `examples/api` (routes → controllers → services → repositories, schemas `requestbody/` + `responsebody/`).
- **CRUD feature** (new, bug fix or refactor) → first read and follow `.ai/workings/00-crud.md`.

Export (capitalise) a name only when another package calls it; helpers used inside one package stay lowercase (`applyUserFilter`, `builder`).

Import `internal/api/presenters`; don't copy an example's `presenters/` folder. Repositories scan into their own models in `internal/api/models/`; `internal/api/models/gen/*.gen.go` is a type reference, never imported. Don't add layers, endpoints or dependencies the example doesn't have.

SQL (repositories only; always `db.Q(ctx)`; never GORM or `database/sql`). Pick one per query:

- **Native SQL** when the statement is fixed (insert, get/delete by id): a string with `$1, $2` placeholders.
- **Squirrel** only when the SQL depends on input (optional filters, optional `SET` columns, paging). Use the package's `sq.Dollar` builder and end with `.ToSql()` → `db.Q(ctx)`; never `RunWith` / Squirrel's `Query` / `Scan`.

Both: values are always bound args, never concatenated into the SQL. List explicit columns (no `SELECT *`) from one shared column list, scanned in that order. Lists `ORDER BY` a unique tiebreaker (`..., id`); a list and its count share one filter helper. `pgx.ErrNoRows` → the repository's not-found sentinel.

Errors: a feature's errors go in `internal/api/exceptions/<feature>.go` (the example's `examples/crud/exceptions/user.go`): plain `errors.New` values, the package importing nothing from the features. Repositories return them, services pass them up unchanged, and the controller returns `fiber.NewError(fiber.StatusInternalServerError, err.Error())` — the message tells the client what went wrong, so write exception messages for clients (`"user not found"`). Unexpected errors (database, driver) go out the same way and can show internal detail; that is accepted for now. Bad input (validation, malformed id) is 400. Never return HTTP statuses below the controller.

Tests: start at the controller layer — HTTP tests like `examples/crud/tests/user_route_test.go`, written to `internal/api/tests/<feature>_route_test.go` with setup from `internal/testsupport`; never import `examples/`. Repository/service tests only when asked.

Before saying a task is done, run `go build ./...`, `go vet ./...` and `go test ./...` (plus `-tags=integration` for the changed feature if Postgres is up). Report failures as they are; never skip or weaken a test to go green.

Responses: an array is `[]`, never `null` — build slices with `make([]T, 0, n)`, never `var s []T`. A nullable field stays a pointer and is sent as `null`; never dereference it to `""` / `0`.
