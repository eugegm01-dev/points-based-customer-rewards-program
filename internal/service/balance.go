package service

import (
	"context"
	"database/sql"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/repository/postgres"
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
// Uses transaction via postgres repository.
func (s *BalanceService) CreditOrder(ctx context.Context, orderNumber string, accrual float64) error {
	pgRepo, ok := s.balanceRepo.(*postgres.BalanceRepository)
	if !ok {
		// fallback для тестов
		order, _ := s.orderRepo.GetByNumber(ctx, orderNumber)
		return s.balanceRepo.Credit(ctx, order.UserID, accrual)
	}

	return pgRepo.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Все запросы в рамках одной транзакции
		return pgRepo.CreditOrderTx(ctx, tx, orderNumber, accrual)
	})
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
