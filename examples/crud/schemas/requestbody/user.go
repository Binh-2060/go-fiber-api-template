package requestbody

/*
Create a user.

Both text columns are here and both are required, which is the rule for every
varchar/text column: the client owns the value, so the API always lets it
supply one.

`name` is `default 'N/A' not null` in the table and is still required. A
default on a text column is a fallback for hand-written SQL inserts, not a
reason to drop the field from the API — leaving it out would produce users
nobody can name at creation, only rename afterwards. Only server-generated
columns stay out of this struct: `id` and `created_at`, which the database
fills and RETURNING reads back.

Both are varchar(200), so the max=200 tags keep an oversized value from
becoming a database error the client cannot read.
*/
type CreateUser struct {
	Name     string `json:"name" form:"name" validate:"required,min=1,max=200"`
	Surename string `json:"surename" form:"surename" validate:"required,min=1,max=200"`
}

/*
Create many users in one request, committed as a single transaction.
*/
type CreateUsers struct {
	Users []CreateUser `json:"users" validate:"required,min=1,max=100,dive"`
}

/*
Update a user. Both fields are required — PATCH here means "replace", not
"merge". The repository's UpdateUser still accepts *string per-column and
skips nils, so a genuinely partial PATCH is one schema change away; this
example just doesn't wire the wire contract that way.
*/
type UpdateUser struct {
	Name     string `json:"name" form:"name" validate:"required,min=1,max=200"`
	Surename string `json:"surename" form:"surename" validate:"required,min=1,max=200"`
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
	PerPage int    `query:"per_page" validate:"omitempty,min=1,max=100"`
	Q       string `query:"q" validate:"omitempty,max=200"`
}
