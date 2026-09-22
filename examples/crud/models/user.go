package models

import "time"

/*
User mirrors the `users` table (see examples/crud/README.md):

	id         uuid                     default gen_random_uuid() not null primary key
	name       varchar(200)             default 'N/A'             not null
	created_at timestamp with time zone default now()             not null
	surename   varchar(200)                                       not null

Field order matches the column list in repositories.userColumns so the same
scan order works for every query.

NOTE: `surename` is spelled that way in the table. The model keeps the
column's spelling rather than silently diverging from the schema — rename the
column first if you want `surname`.
*/
type User struct {
	ID        string
	Name      string
	Surename  string
	CreatedAt time.Time
}
