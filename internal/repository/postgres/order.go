// Package postgres implements domain interfaces using PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"errors"

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
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at
		FROM orders
		WHERE status = $1`,
		domain.OrderStatusNew,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
			&order.ProcessedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// Create inserts a new order and sets order.ID. Returns domain.ErrDuplicateOrder if number exists.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO orders (user_id, number, status, uploaded_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		order.UserID, order.Number, order.Status, order.UploadedAt,
	).Scan(&order.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return domain.ErrDuplicateOrder
		}
		return err
	}

	return nil
}

// GetByNumber fetches an order by number. Returns domain.ErrOrderNotFound if not found.
func (r *OrderRepository) GetByNumber(ctx context.Context, number string) (*domain.Order, error) {
	order := &domain.Order{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at
		FROM orders WHERE number = $1`,
		number,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&order.Accrual,
		&order.UploadedAt,
		&order.ProcessedAt,
	)

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
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
			&order.ProcessedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

// UpdateStatus updates order status and optionally accrual amount.
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID int64, status domain.OrderStatus, accrual *float64) error {
	query := `UPDATE orders SET status = $1`
	args := []interface{}{status}

	if accrual != nil {
		query += `, accrual = $2`
		args = append(args, *accrual)
	}

	// Set processed_at for terminal states
	terminalStates := map[domain.OrderStatus]bool{
		domain.OrderStatusProcessed: true,
		domain.OrderStatusInvalid:   true,
	}
	if terminalStates[status] {
		query += `, processed_at = NOW()`
	}

	query += ` WHERE id = $` + string(rune(len(args)+48+1)) // ASCII '1' = 49
	args = append(args, orderID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetProcessingOrders fetches orders with PROCESSING status for background worker.
func (r *OrderRepository) GetProcessingOrders(ctx context.Context) ([]*domain.Order, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, number, status, accrual, uploaded_at, processed_at
		FROM orders
		WHERE status = $1`,
		domain.OrderStatusProcessing,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
			&order.ProcessedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
