package models

/*
User is an in-memory account record.

This example has no migration and no repository backed by pkg/db — it exists
to demonstrate pkg/jwt's RSAManager, not another CRUD. PasswordHash is a
bcrypt hash, never the plaintext password.
*/
type User struct {
	ID           string
	Email        string
	PasswordHash string
}
