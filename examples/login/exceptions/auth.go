/*
Package exceptions lists every error the login feature can return on purpose.

Repositories return these values, services pass them up unchanged, and the
controller sends err.Error() as the response message. The package imports
nothing from the feature, so any layer can use it without an import cycle.
*/
package exceptions

import "errors"

// ErrUserNotFound is returned when no account matches the given email.
var ErrUserNotFound = errors.New("user not found")

/*
ErrInvalidCredentials covers both "no such user" and "wrong password".

A caller must not be able to tell the two apart — that's what lets an
attacker enumerate registered emails one login attempt at a time.
*/
var ErrInvalidCredentials = errors.New("invalid email or password")
