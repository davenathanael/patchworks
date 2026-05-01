package auth

// User represents an authenticated user in the auth system.
// All IDs are opaque strings to keep pkg/auth decoupled from internal domain types.
type User struct {
	ID    string
	Email string
}
