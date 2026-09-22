/*
Package exceptions lists every error the users feature can return on purpose.

Repositories return these values, services pass them up unchanged, and the
controller sends err.Error() as the response message. The package imports
nothing from the feature, so any layer can use it without an import cycle.

Go has no exceptions: these are ordinary error values, checked with errors.Is.
*/
package exceptions

import "errors"

// ErrUserNotFound is returned when a query matched no row.
var ErrUserNotFound = errors.New("user not found")

/*
ErrNoUpdateFields is returned when an update was asked to change nothing. An
UPDATE with an empty SET clause is not valid SQL, so this is caught before the
query is built.
*/
var ErrNoUpdateFields = errors.New("no fields to update")
