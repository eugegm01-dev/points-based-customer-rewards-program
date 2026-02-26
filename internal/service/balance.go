// Package service contains application business logic (use cases).
package service

import (
	"context"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
)

type BalanceService struct {
	balanceRepo domain.BalanceRepository
	orderRepo   domain.OrderRepository
}

func NewBalanceService(balanceRepo domain.BalanceRepository, orderRepo domain.OrderRepository) *BalanceService {
	return &BalanceService{balanceRepo: balanceRepo, orderRepo: orderRepo}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID string) (*domain.Balance, error) {
	return s.balanceRepo.GetOrCreate(ctx, userID)
}

// CreditOrder credits points from a processed order to user's balance.
func (s *BalanceService) CreditOrder(ctx context.Context, orderNumber string, accrual float64) error {
	order, err := s.orderRepo.GetByNumber(ctx, orderNumber)
	if err != nil {
		return err
	}
	return s.balanceRepo.Credit(ctx, order.UserID, accrual)
}

func (s *BalanceService) WithdrawRequest(ctx context.Context, userID, orderNumber string, sum float64) error {
	if !domain.IsValidNumber(orderNumber) {
		return ErrInvalidOrderNumber
	}
	_, err := s.balanceRepo.Withdraw(ctx, userID, orderNumber, sum)
	return err
}

func (s *BalanceService) GetWithdrawals(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
	return s.balanceRepo.GetWithdrawals(ctx, userID)
}
