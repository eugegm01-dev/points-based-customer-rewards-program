// Package domain defines business entities and interfaces.
package domain

import (
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

// Sentinel errors for balance operations.
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)
