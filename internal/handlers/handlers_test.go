package handlers_test

import (
	"context"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
)

// mockAuthService
type mockAuthService struct {
	registerFunc func(ctx context.Context, login, password string) (*domain.User, error)
	loginFunc    func(ctx context.Context, login, password string) (*domain.User, error)
}

func (m *mockAuthService) Register(ctx context.Context, login, password string) (*domain.User, error) {
	return m.registerFunc(ctx, login, password)
}
func (m *mockAuthService) Login(ctx context.Context, login, password string) (*domain.User, error) {
	return m.loginFunc(ctx, login, password)
}

// mockOrderService
type mockOrderService struct {
	uploadOrderFunc     func(ctx context.Context, userID, number string) error
	getOrdersByUserFunc func(ctx context.Context, userID string) ([]*domain.Order, error)
}

func (m *mockOrderService) UploadOrder(ctx context.Context, userID, number string) error {
	return m.uploadOrderFunc(ctx, userID, number)
}
func (m *mockOrderService) GetOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	return m.getOrdersByUserFunc(ctx, userID)
}

// mockBalanceService
type mockBalanceService struct {
	getBalanceFunc      func(ctx context.Context, userID string) (*domain.Balance, error)
	withdrawRequestFunc func(ctx context.Context, userID, orderNumber string, sum float64) error
	getWithdrawalsFunc  func(ctx context.Context, userID string) ([]*domain.Withdrawal, error)
}

func (m *mockBalanceService) GetBalance(ctx context.Context, userID string) (*domain.Balance, error) {
	return m.getBalanceFunc(ctx, userID)
}
func (m *mockBalanceService) WithdrawRequest(ctx context.Context, userID, orderNumber string, sum float64) error {
	return m.withdrawRequestFunc(ctx, userID, orderNumber, sum)
}
func (m *mockBalanceService) GetWithdrawals(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
	return m.getWithdrawalsFunc(ctx, userID)
}
