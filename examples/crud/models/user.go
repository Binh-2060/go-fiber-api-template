package models

/*
User is the repository's own model: the columns its queries return, typed from
the reference internal/api/models/gen/users.gen.go (never imported):

	id   uuid         default gen_random_uuid() not null primary key
	name varchar(255)                           not null

A model is shaped by its queries, not by the table — columns the example never
reads are left out. Field order matches repositories.userColumns so the same
scan order works for every query. Re-check it against the reference after
`go run ./cmd/gorm`.
*/
type User struct {
	ID   string
	Name string
}
