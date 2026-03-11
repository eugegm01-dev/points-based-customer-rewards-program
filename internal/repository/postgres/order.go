// Package postgres implements domain interfaces using PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
)

// OrderRepository implements domain.OrderRepository for PostgreSQL.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository returns a new OrderRepository.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// GetNewOrders fetches orders with NEW status for background worker.
func (r *OrderRepository) GetNewOrders(ctx context.Context) ([]*domain.Order, error) {
	orders, err := WithRetry(ctx, 3, 100*time.Millisecond, func() ([]*domain.Order, error) {
		rows, err := r.db.QueryContext(ctx,
			`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at FROM orders WHERE status = $1`,
			domain.OrderStatusNew,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var ords []*domain.Order
		for rows.Next() {
			var o domain.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt, &o.ProcessedAt); err != nil {
				return nil, err
			}
			ords = append(ords, &o)
		}
		return ords, rows.Err()
	})
	return orders, err
}

// Create inserts a new order and sets order.ID. Returns domain.ErrDuplicateOrder if number exists.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	err := Retry(ctx, 3, 100*time.Millisecond, func() error {
		return r.db.QueryRowContext(ctx,
			`INSERT INTO orders (user_id, number, status, uploaded_at) VALUES ($1, $2, $3, $4) RETURNING id`,
			order.UserID, order.Number, order.Status, order.UploadedAt,
		).Scan(&order.ID)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateOrder
		}
		return err
	}
	return nil
}

// GetByNumber fetches an order by number. Returns domain.ErrOrderNotFound if not found.
func (r *OrderRepository) GetByNumber(ctx context.Context, number string) (*domain.Order, error) {
	order, err := WithRetry(ctx, 3, 100*time.Millisecond, func() (*domain.Order, error) {
		var o domain.Order
		err := r.db.QueryRowContext(ctx,
			`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at FROM orders WHERE number = $1`,
			number,
		).Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt, &o.ProcessedAt)
		return &o, err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

// GetByUserID fetches all orders for a user, sorted by uploaded_at DESC.
func (r *OrderRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	orders, err := WithRetry(ctx, 3, 100*time.Millisecond, func() ([]*domain.Order, error) {
		rows, err := r.db.QueryContext(ctx,
			`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`,
			userID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var ords []*domain.Order
		for rows.Next() {
			var o domain.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt, &o.ProcessedAt); err != nil {
				return nil, err
			}
			ords = append(ords, &o)
		}
		return ords, rows.Err()
	})
	return orders, err
}

// UpdateStatus updates order status and optionally accrual amount.

func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error {
	return Retry(ctx, 3, 100*time.Millisecond, func() error {
		query := `UPDATE orders SET status = $1`
		args := []interface{}{status}
		if accrual != nil {
			query += `, accrual = $2`
			args = append(args, *accrual)
		}
		if status == domain.OrderStatusProcessed || status == domain.OrderStatusInvalid {
			query += `, processed_at = NOW()`
		}
		query += ` WHERE id = $` + strconv.Itoa(len(args)+1)
		args = append(args, orderID)
		_, err := r.db.ExecContext(ctx, query, args...)
		return err
	})
}

// GetProcessingOrders fetches orders with PROCESSING status for background worker.
func (r *OrderRepository) GetProcessingOrders(ctx context.Context) ([]*domain.Order, error) {
	orders, err := WithRetry(ctx, 3, 100*time.Millisecond, func() ([]*domain.Order, error) {
		rows, err := r.db.QueryContext(ctx,
			`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at FROM orders WHERE status = $1`,
			domain.OrderStatusProcessing,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var ords []*domain.Order
		for rows.Next() {
			var o domain.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt, &o.ProcessedAt); err != nil {
				return nil, err
			}
			ords = append(ords, &o)
		}
		return ords, rows.Err()
	})
	return orders, err
}
