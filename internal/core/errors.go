package core

import "errors"

// Sentinel errors classify expected, user-facing failures. Handlers and the
// HTTP error wrapper branch on them with errors.Is; the db/auth layers wrap
// underlying errors with these sentinels so framework types (e.g. pgconn)
// never cross layer boundaries.
var (
	// ErrEmailTaken is returned when a user with the given email already exists.
	ErrEmailTaken = errors.New("email already registered")

	// ErrInvalidCredentials is returned when login fails due to a bad email or password.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrForbidden is returned when a user lacks the role needed for an action.
	ErrForbidden = errors.New("forbidden")
)
