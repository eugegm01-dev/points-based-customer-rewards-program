// Package domain defines business entities and interfaces.
// It has no external dependencies and forms the core of the application.
package domain

import (
	"time"
)

// OrderStatus represents the processing state of an order.
type OrderStatus string

const (
	// OrderStatusNew means the order was uploaded but not yet sent to accrual system.
	OrderStatusNew OrderStatus = "NEW"
	// OrderStatusProcessing means the accrual system is calculating rewards.
	OrderStatusProcessing OrderStatus = "PROCESSING"
	// OrderStatusInvalid means the accrual system rejected the order.
	OrderStatusInvalid OrderStatus = "INVALID"
	// OrderStatusProcessed means rewards were successfully calculated and credited.
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

// Order represents a user-uploaded order number for loyalty points calculation.
type Order struct {
	ID          int64  // SERIAL primary key
	UserID      string // UUID reference to users(id)
	Number      string // Unique order number (Luhn-validated)
	Status      OrderStatus
	Accrual     *float64   // Nullable - points awarded (1 point = 1 ruble)
	UploadedAt  time.Time  // When user uploaded the order
	ProcessedAt *time.Time // When status became PROCESSED or INVALID
}

// IsValidNumber checks if the order number passes Luhn algorithm validation.
func IsValidNumber(number string) bool {
	if len(number) < 2 {
		return false
	}

	sum := 0
	alt := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}

		if alt {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alt = !alt
	}

	return sum%10 == 0
}
