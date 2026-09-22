package requestbody

/*
Create a user.

Derived from internal/api/models/gen/users.gen.go (see .ai/workings/00-crud.md):
`name` is `not null`, so it is required, and `varchar(255)`, so max=255 keeps an
oversized value from becoming a database error the client cannot read. `id` is
the primary key with a gen_random_uuid() default — the database fills it and
RETURNING reads it back, so it stays out of this struct.
*/
type CreateUser struct {
	Name string `json:"name" form:"name" validate:"required,max=255"`
}

/*
Update a user. Same rules as CreateUser: `name` is the only client-owned column,
so it is required. The repository's UpdateUser still accepts *string per column
and skips nils, so a genuinely partial update is one schema change away once the
table has more columns.
*/
type UpdateUser struct {
	Name string `json:"name" form:"name" validate:"required,max=255"`
}

/*
Query string for the list endpoint.

Zero values mean "not supplied" and the service substitutes its defaults. Fiber
v3 binds these through `query` tags via c.Bind().Query — v2's QueryInt and
friends are gone.

Q is an optional case-insensitive substring filter on name. It is what makes
the list query dynamic, and therefore what the Squirrel builder in the
repository exists for.
*/
type ListUsers struct {
	Page    int    `query:"page" validate:"omitempty,min=1"`
	PerPage int    `query:"perPage" validate:"omitempty,min=1,max=100"`
	Q       string `query:"q" validate:"omitempty,max=255"`
}
