// Package postgres implements domain interfaces using PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
)

type BalanceRepository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) GetOrCreate(ctx context.Context, userID string) (*domain.Balance, error) {
	b := &domain.Balance{}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO balances (user_id, current, withdrawn, updated_at)
		VALUES ($1, 0, 0, now())
		ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		RETURNING user_id, current, withdrawn, updated_at`,
		userID,
	).Scan(&b.UserID, &b.Current, &b.Withdrawn, &b.UpdatedAt)
	return b, err
}

func (r *BalanceRepository) Credit(ctx context.Context, userID string, amount float64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO balances (user_id, current, withdrawn, updated_at)
                VALUES ($1, $2, 0, now())
                ON CONFLICT (user_id) DO UPDATE
                SET current = balances.current + $2, updated_at = now()`,
		userID, amount,
	)
	return err
}

func (r *BalanceRepository) Withdraw(ctx context.Context, userID string, orderNumber string, sum float64) (*domain.Withdrawal, error) {
	// Atomic withdrawal: only succeed if current >= sum
	result, err := r.db.ExecContext(ctx,
		`UPDATE balances 
		SET current = current - $1, withdrawn = withdrawn + $1, updated_at = now()
		WHERE user_id = $2 AND current >= $1`,
		sum, userID,
	)
	if err != nil {
		return nil, err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return nil, domain.ErrInsufficientFunds
	}

	w := &domain.Withdrawal{
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum, processed_at)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		w.UserID, w.OrderNumber, w.Sum, w.ProcessedAt,
	).Scan(&w.ID)
	return w, err
}

func (r *BalanceRepository) GetWithdrawals(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, order_number, sum, processed_at 
		FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ws []*domain.Withdrawal
	for rows.Next() {
		w := &domain.Withdrawal{}
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}
	return ws, rows.Err()
}
