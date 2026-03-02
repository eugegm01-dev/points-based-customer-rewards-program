package service

import (
	"context"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/stretchr/testify/assert"
)

type mockOrderRepo struct {
	createFunc              func(ctx context.Context, order *domain.Order) error
	getByNumberFunc         func(ctx context.Context, number string) (*domain.Order, error)
	getByUserIDFunc         func(ctx context.Context, userID string) ([]*domain.Order, error)
	updateStatusFunc        func(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error
	getProcessingOrdersFunc func(ctx context.Context) ([]*domain.Order, error)
	getNewOrdersFunc        func(ctx context.Context) ([]*domain.Order, error)
}

func (m *mockOrderRepo) Create(ctx context.Context, order *domain.Order) error {
	return m.createFunc(ctx, order)
}
func (m *mockOrderRepo) GetByNumber(ctx context.Context, number string) (*domain.Order, error) {
	return m.getByNumberFunc(ctx, number)
}
func (m *mockOrderRepo) GetByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	return m.getByUserIDFunc(ctx, userID)
}
func (m *mockOrderRepo) UpdateStatus(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error {
	return m.updateStatusFunc(ctx, orderID, status, accrual)
}
func (m *mockOrderRepo) GetProcessingOrders(ctx context.Context) ([]*domain.Order, error) {
	return m.getProcessingOrdersFunc(ctx)
}
func (m *mockOrderRepo) GetNewOrders(ctx context.Context) ([]*domain.Order, error) {
	return m.getNewOrdersFunc(ctx)
}

func TestOrderService_UploadOrder(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		number          string
		mockGetByNumber func(ctx context.Context, number string) (*domain.Order, error)
		mockCreate      func(ctx context.Context, order *domain.Order) error
		wantErr         error
	}{
		{
			name:   "success",
			userID: "u1",
			number: "12345678903",
			mockGetByNumber: func(ctx context.Context, number string) (*domain.Order, error) {
				return nil, domain.ErrOrderNotFound
			},
			mockCreate: func(ctx context.Context, order *domain.Order) error {
				order.ID = 1
				return nil
			},
			wantErr: nil,
		},
		{
			name:    "invalid number",
			userID:  "u1",
			number:  "123",
			wantErr: ErrInvalidOrderNumber,
		},
		{
			name:   "duplicate own order",
			userID: "u1",
			number: "12345678903",
			mockGetByNumber: func(ctx context.Context, number string) (*domain.Order, error) {
				return &domain.Order{UserID: "u1", Number: number}, nil
			},
			wantErr: ErrDuplicateOrder,
		},
		{
			name:   "duplicate other user",
			userID: "u2",
			number: "12345678903",
			mockGetByNumber: func(ctx context.Context, number string) (*domain.Order, error) {
				return &domain.Order{UserID: "u1", Number: number}, nil
			},
			wantErr: ErrOrderBelongsToAnotherUser,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrderRepo{
				getByNumberFunc: tt.mockGetByNumber,
				createFunc:      tt.mockCreate,
			}
			svc := NewOrderService(repo)
			err := svc.UploadOrder(context.Background(), tt.userID, tt.number)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
func TestOrderService_GetOrdersByUserID(t *testing.T) {
	repo := &mockOrderRepo{
		getByUserIDFunc: func(ctx context.Context, userID string) ([]*domain.Order, error) {
			return []*domain.Order{{Number: "111"}}, nil
		},
	}
	svc := NewOrderService(repo)
	orders, err := svc.GetOrdersByUserID(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, "111", orders[0].Number)
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	repo := &mockOrderRepo{
		updateStatusFunc: func(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error {
			return nil
		},
	}
	svc := NewOrderService(repo)
	err := svc.UpdateOrderStatus(context.Background(), 1, domain.OrderStatusProcessed, nil)
	assert.NoError(t, err)
}

func TestOrderService_GetProcessingOrders(t *testing.T) {
	repo := &mockOrderRepo{
		getProcessingOrdersFunc: func(ctx context.Context) ([]*domain.Order, error) {
			return []*domain.Order{{Number: "111"}}, nil
		},
	}
	svc := NewOrderService(repo)
	orders, err := svc.GetProcessingOrders(context.Background())
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
}

func TestOrderService_GetNewOrders(t *testing.T) {
	repo := &mockOrderRepo{
		getNewOrdersFunc: func(ctx context.Context) ([]*domain.Order, error) {
			return []*domain.Order{{Number: "111"}}, nil
		},
	}
	svc := NewOrderService(repo)
	orders, err := svc.GetNewOrders(context.Background())
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
}
func TestOrderService_GetOrderByNumber(t *testing.T) {
	repo := &mockOrderRepo{
		getByNumberFunc: func(ctx context.Context, number string) (*domain.Order, error) {
			return &domain.Order{Number: number}, nil
		},
	}
	svc := NewOrderService(repo)
	order, err := svc.GetOrderByNumber(context.Background(), "123")
	assert.NoError(t, err)
	assert.Equal(t, "123", order.Number)
}

func TestOrderService_GetOrderByNumber_NotFound(t *testing.T) {
	repo := &mockOrderRepo{
		getByNumberFunc: func(ctx context.Context, number string) (*domain.Order, error) {
			return nil, domain.ErrOrderNotFound
		},
	}
	svc := NewOrderService(repo)
	_, err := svc.GetOrderByNumber(context.Background(), "123")
	assert.ErrorIs(t, err, domain.ErrOrderNotFound)
}

func TestOrderService_UpdateOrderStatus_Error(t *testing.T) {
	repo := &mockOrderRepo{
		updateStatusFunc: func(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error {
			return assert.AnError
		},
	}
	svc := NewOrderService(repo)
	err := svc.UpdateOrderStatus(context.Background(), 1, domain.OrderStatusProcessed, nil)
	assert.Error(t, err)
}
