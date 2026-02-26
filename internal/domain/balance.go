// Package domain defines business entities and interfaces.
package domain

import (
	"context"
	"errors"
	"time"
)

// Balance tracks a user's loyalty points: current available + total withdrawn.
type Balance struct {
	UserID    string
	Current   float64
	Withdrawn float64
	UpdatedAt time.Time
}

// Withdrawal represents a user's request to spend loyalty points.
type Withdrawal struct {
	ID          int64
	UserID      string
	OrderNumber string
	Sum         float64
	ProcessedAt time.Time
}

// BalanceRepository defines operations for balance persistence.
type BalanceRepository interface {
	GetOrCreate(ctx context.Context, userID string) (*Balance, error)
	Credit(ctx context.Context, userID string, amount float64) error
	Withdraw(ctx context.Context, userID string, orderNumber string, sum float64) (*Withdrawal, error)
	GetWithdrawals(ctx context.Context, userID string) ([]*Withdrawal, error)
}

// Sentinel errors for balance operations.
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)
