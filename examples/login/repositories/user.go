package repositories

import (
	"context"
	"errors"

	"github.com/Binh-2060/go-application-template/examples/login/models"
	"golang.org/x/crypto/bcrypt"
)

// ErrUserNotFound is returned when no account matches the given email.
var ErrUserNotFound = errors.New("repositories: user not found")

/*
users is a hardcoded in-memory store standing in for pkg/db.

A real feature resolves its connection with db.Q(ctx) the way
examples/crud/repositories does; this example only exists to show
pkg/jwt.RSAManager, so a map is enough and keeps it runnable with no
database and no migration.
*/
var users = map[string]models.User{}

// seedUser hashes password once, at package init, rather than committing a
// bcrypt hash literal — the literal would tie this file to one bcrypt cost
// and be indistinguishable from a real credential at a glance.
func seedUser(id, email, password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic("examples/login: seed user: " + err.Error())
	}

	users[email] = models.User{ID: id, Email: email, PasswordHash: string(hash)}
}

func init() {
	seedUser("00000000-0000-0000-0000-000000000001", "demo@example.com", "password123")
}

// GetUserByEmail looks up a user by email, or ErrUserNotFound.
func GetUserByEmail(_ context.Context, email string) (models.User, error) {
	user, ok := users[email]
	if !ok {
		return models.User{}, ErrUserNotFound
	}

	return user, nil
}
