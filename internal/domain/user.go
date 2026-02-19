// Package domain defines business entities and interfaces.
// It has no external dependencies and forms the core of the application.
package domain

import "time"

// User represents a registered user in the loyalty system.
// Fields correspond to the users table in the database.
type User struct {
	ID           string    // UUID, primary key
	Login        string    // Unique login (username)
	PasswordHash string    // bcrypt hash of password, never exposed
	CreatedAt    time.Time // Registration timestamp
}
