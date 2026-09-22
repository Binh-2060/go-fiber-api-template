# CRUD workflow

1. Check the reference exists: `ls internal/api/models/gen | grep -x <table>.gen.go`
   - Found → it is the real schema: Go type, `type:` tag (SQL type, `max=` limits) and nullability per column. Read it; never import or edit it.
   - Missing, table exists → `go run ./cmd/gorm`, check again.
   - Missing, no table → stop and ask the user for the schema.
2. Write the repository's own model in `internal/api/models/<table>.go`: the columns its queries return, types copied from the reference. Nullable column = pointer field: scanned as nil, kept a pointer through to the response, sent as `null`. `examples/crud/models/user.go` is an illustration only — never copy its fields or limits.
3. Request body validators (`schemas/requestbody/`), from each field's `gorm` tag:
   - `not null` → `required` (text columns with a default too)
   - `primaryKey` / server default (`gen_random_uuid()`, `now()`) → leave out of the body
   - `not null` bool/number where `false`/`0` is valid → pointer + `required`
   - nullable → pointer + `omitempty`
   - `varchar(N)` → add `max=N`; `uuid` → add `uuid`

   ```go
   // gen/users.gen.go: ID string `gorm:"...;type:uuid;primaryKey;default:gen_random_uuid()"`
   //                   Name string `gorm:"...;type:character varying(255);not null"`
   type CreateUser struct {
   	Name string `json:"name" form:"name" validate:"required,max=255"` // ID left out
   }
   ```

   Tags only run through `validators.ParseAndValidateBody` / `ParseAndValidateQueryParam`.
4. Copy the code in `examples/crud` (routes, controllers, services, repositories, schemas, exceptions) into the matching `internal/api/` folders: same names, style. Tests: controller tests only, copying `tests/user_route_test.go` into `internal/api/tests/<table>_route_test.go` (add `main_test.go` with `testsupport.Main` if the folder is new). Data access via `pkg/db` + `db.Q(ctx)`, never GORM. Open `examples/crud/README.md` only if unsure why the code does something.
