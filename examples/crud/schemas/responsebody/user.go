package responsebody

import (
	"github.com/Binh-2060/go-application-template/examples/crud/models"
)

/*
User is the wire shape of a user.

Deliberately a separate type from models.User: the model tracks the table, this
tracks the API contract. Adding a column then stays a decision about what to
expose rather than an accidental leak, and a column rename does not silently
break every client.
*/
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

/*
Map a model to its response shape.
*/
func NewUser(m models.User) User {
	return User{
		ID:   m.ID,
		Name: m.Name,
	}
}

/*
Map a slice of models, preserving order.

Returns an empty non-nil slice for empty input so the JSON is [] and not null.
*/
func NewUsers(ms []models.User) []User {
	users := make([]User, 0, len(ms))
	for _, m := range ms {
		users = append(users, NewUser(m))
	}

	return users
}
