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

// OrderRepository defines operations for order persistence.
type OrderRepository interface {
	// Create inserts a new order. Returns ErrDuplicateOrder if number exists.
	Create(ctx context.Context, order *Order) error
	// GetByNumber fetches an order by number. Returns ErrOrderNotFound if not found.
	GetByNumber(ctx context.Context, number string) (*Order, error)
	// GetByUserID fetches all orders for a user, sorted by uploaded_at DESC.
	GetByUserID(ctx context.Context, userID string) ([]*Order, error)
	// UpdateStatus updates order status and optionally accrual amount.
	UpdateStatus(ctx context.Context, orderID int64, status OrderStatus, accrual *float64) error
	// GetProcessingOrders fetches orders with PROCESSING status for background worker.
	GetProcessingOrders(ctx context.Context) ([]*Order, error)
	// GetNewOrders fetches orders with NEW status for background worker.
	GetNewOrders(ctx context.Context) ([]*Order, error)
}

// Sentinel errors for repository operations.
// Callers can use errors.Is(err, domain.ErrXXX) to handle specific cases.
var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateLogin = errors.New("login already exists")
	ErrOrderNotFound  = errors.New("order not found")
	ErrDuplicateOrder = errors.New("order number already exists")
)
