// Package service contains application business logic (use cases).
package service

import (
	"context"
	"errors"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
)

// OrderService handles order upload, retrieval, and business rules.
type OrderService struct {
	orderRepo domain.OrderRepository
}

// NewOrderService returns a new OrderService.
func NewOrderService(orderRepo domain.OrderRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo}
}

// UploadOrder processes a user-submitted order number.
func (s *OrderService) UploadOrder(ctx context.Context, userID, number string) error {
	if !domain.IsValidNumber(number) {
		return ErrInvalidOrderNumber
	}

	existing, err := s.orderRepo.GetByNumber(ctx, number)
	if err != nil {
		if !errors.Is(err, domain.ErrOrderNotFound) {
			return err
		}
		// Order not found → proceed to create
	} else {
		if existing.UserID == userID {
			return ErrDuplicateOrder
		}
		return ErrOrderBelongsToAnotherUser
	}

	order := &domain.Order{
		UserID:     userID,
		Number:     number,
		Status:     domain.OrderStatusNew,
		UploadedAt: time.Now(),
	}

	return s.orderRepo.Create(ctx, order)
}

// GetOrdersByUserID returns all orders for a user, sorted by uploaded_at DESC.
func (s *OrderService) GetOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	return s.orderRepo.GetByUserID(ctx, userID)
}

// GetOrderByNumber returns an order by number (for accrual worker).
func (s *OrderService) GetOrderByNumber(ctx context.Context, number string) (*domain.Order, error) {
	return s.orderRepo.GetByNumber(ctx, number)
}

// UpdateOrderStatus updates order status and optionally accrual amount.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error {
	return s.orderRepo.UpdateStatus(ctx, orderID, status, accrual)
}

// GetProcessingOrders returns ALL orders with PROCESSING status (for background worker).
// Note: This is NOT per-user — the accrual worker needs to check all processing orders globally.
func (s *OrderService) GetProcessingOrders(ctx context.Context) ([]*domain.Order, error) {
	return s.orderRepo.GetProcessingOrders(ctx) // ✅ NO userID PARAMETER
}

// Sentinel errors for order operations.
var (
	ErrInvalidOrderNumber        = errors.New("invalid order number format")
	ErrDuplicateOrder            = errors.New("order already uploaded by this user")
	ErrOrderBelongsToAnotherUser = errors.New("order already uploaded by another user")
)
