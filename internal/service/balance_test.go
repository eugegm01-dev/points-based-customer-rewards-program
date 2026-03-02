package service

import (
	"context"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/stretchr/testify/assert"
)

type mockBalanceRepo struct {
	getOrCreateFunc    func(ctx context.Context, userID string) (*domain.Balance, error)
	creditFunc         func(ctx context.Context, userID string, amount float64) error
	withdrawFunc       func(ctx context.Context, userID string, orderNumber string, sum float64) (*domain.Withdrawal, error)
	getWithdrawalsFunc func(ctx context.Context, userID string) ([]*domain.Withdrawal, error)
}

func (m *mockBalanceRepo) GetOrCreate(ctx context.Context, userID string) (*domain.Balance, error) {
	return m.getOrCreateFunc(ctx, userID)
}
func (m *mockBalanceRepo) Credit(ctx context.Context, userID string, amount float64) error {
	return m.creditFunc(ctx, userID, amount)
}
func (m *mockBalanceRepo) Withdraw(ctx context.Context, userID string, orderNumber string, sum float64) (*domain.Withdrawal, error) {
	return m.withdrawFunc(ctx, userID, orderNumber, sum)
}
func (m *mockBalanceRepo) GetWithdrawals(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
	return m.getWithdrawalsFunc(ctx, userID)
}

func TestBalanceService_WithdrawRequest(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		orderNumber  string
		sum          float64
		mockWithdraw func(ctx context.Context, userID string, orderNumber string, sum float64) (*domain.Withdrawal, error)
		wantErr      error
	}{
		{
			name:        "success",
			userID:      "u1",
			orderNumber: "12345678903",
			sum:         100,
			mockWithdraw: func(ctx context.Context, userID string, orderNumber string, sum float64) (*domain.Withdrawal, error) {
				return &domain.Withdrawal{}, nil
			},
			wantErr: nil,
		},
		{
			name:        "invalid order number",
			userID:      "u1",
			orderNumber: "123",
			sum:         100,
			wantErr:     ErrInvalidOrderNumber,
		},
		{
			name:        "insufficient funds",
			userID:      "u1",
			orderNumber: "12345678903",
			sum:         100,
			mockWithdraw: func(ctx context.Context, userID string, orderNumber string, sum float64) (*domain.Withdrawal, error) {
				return nil, domain.ErrInsufficientFunds
			},
			wantErr: domain.ErrInsufficientFunds,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balanceRepo := &mockBalanceRepo{withdrawFunc: tt.mockWithdraw}
			orderRepo := &mockOrderRepo{} // not used in this test
			svc := NewBalanceService(balanceRepo, orderRepo)
			err := svc.WithdrawRequest(context.Background(), tt.userID, tt.orderNumber, tt.sum)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestBalanceService_GetBalance(t *testing.T) {
	balanceRepo := &mockBalanceRepo{
		getOrCreateFunc: func(ctx context.Context, userID string) (*domain.Balance, error) {
			return &domain.Balance{UserID: userID, Current: 100, Withdrawn: 10}, nil
		},
	}
	orderRepo := &mockOrderRepo{}
	svc := NewBalanceService(balanceRepo, orderRepo)
	bal, err := svc.GetBalance(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Equal(t, 100.0, bal.Current)
	assert.Equal(t, 10.0, bal.Withdrawn)
}

func TestBalanceService_CreditOrder(t *testing.T) {
	orderRepo := &mockOrderRepo{
		getByNumberFunc: func(ctx context.Context, number string) (*domain.Order, error) {
			return &domain.Order{UserID: "u1", Number: number}, nil
		},
	}
	balanceRepo := &mockBalanceRepo{
		creditFunc: func(ctx context.Context, userID string, amount float64) error {
			return nil
		},
	}
	svc := NewBalanceService(balanceRepo, orderRepo)
	err := svc.CreditOrder(context.Background(), "123", 50)
	assert.NoError(t, err)
}

func TestBalanceService_GetWithdrawals(t *testing.T) {
	balanceRepo := &mockBalanceRepo{
		getWithdrawalsFunc: func(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
			return []*domain.Withdrawal{{ID: 1, Sum: 10}}, nil
		},
	}
	orderRepo := &mockOrderRepo{}
	svc := NewBalanceService(balanceRepo, orderRepo)
	ws, err := svc.GetWithdrawals(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Len(t, ws, 1)
	assert.Equal(t, 10.0, ws[0].Sum)
}
func TestBalanceService_CreditOrder_OrderNotFound(t *testing.T) {
	orderRepo := &mockOrderRepo{
		getByNumberFunc: func(ctx context.Context, number string) (*domain.Order, error) {
			return nil, domain.ErrOrderNotFound
		},
	}
	balanceRepo := &mockBalanceRepo{}
	svc := NewBalanceService(balanceRepo, orderRepo)
	err := svc.CreditOrder(context.Background(), "123", 50)
	assert.ErrorIs(t, err, domain.ErrOrderNotFound)
}

func TestBalanceService_CreditOrder_CreditFails(t *testing.T) {
	orderRepo := &mockOrderRepo{
		getByNumberFunc: func(ctx context.Context, number string) (*domain.Order, error) {
			return &domain.Order{UserID: "u1"}, nil
		},
	}
	balanceRepo := &mockBalanceRepo{
		creditFunc: func(ctx context.Context, userID string, amount float64) error {
			return assert.AnError
		},
	}
	svc := NewBalanceService(balanceRepo, orderRepo)
	err := svc.CreditOrder(context.Background(), "123", 50)
	assert.Error(t, err)
}
func TestBalanceService_GetBalance_RepoError(t *testing.T) {
	balanceRepo := &mockBalanceRepo{
		getOrCreateFunc: func(ctx context.Context, userID string) (*domain.Balance, error) {
			return nil, assert.AnError
		},
	}
	orderRepo := &mockOrderRepo{}
	svc := NewBalanceService(balanceRepo, orderRepo)
	_, err := svc.GetBalance(context.Background(), "u1")
	assert.Error(t, err)
}
