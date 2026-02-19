package domain

import (
	"context"
	"errors"
)

// UserRepository defines operations for user persistence.
// The interface is declared in domain so that services depend on abstraction,
// not on a specific DB implementation. This enables testing with mocks.
type UserRepository interface {
	// Create inserts a new user. Returns ErrDuplicateLogin if login already exists.
	Create(ctx context.Context, user *User) error

	// GetByLogin fetches a user by login. Returns ErrUserNotFound if not found.
	GetByLogin(ctx context.Context, login string) (*User, error)
}

// Sentinel errors for repository operations.
// Callers can use errors.Is(err, domain.ErrUserNotFound) to handle specific cases.
var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateLogin = errors.New("login already exists")
)
