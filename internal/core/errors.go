package core

import "errors"

// ErrEmailTaken is returned when a user with the given email already exists.
var ErrEmailTaken = errors.New("email already registered")
