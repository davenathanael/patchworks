package core

import "github.com/google/uuid"

// User represents one user of the application.
type User struct {
	ID    uuid.UUID
	Email string
}
