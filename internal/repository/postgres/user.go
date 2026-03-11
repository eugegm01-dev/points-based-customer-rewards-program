// Package postgres implements domain interfaces using PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

// UserRepository implements domain.UserRepository for PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository returns a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user and sets u.ID. Returns domain.ErrDuplicateLogin if login already exists.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	err := Retry(ctx, 3, 100*time.Millisecond, func() error {
		return r.db.QueryRowContext(ctx,
			`INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, created_at`,
			u.Login, u.PasswordHash,
		).Scan(&u.ID, &u.CreatedAt)
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateLogin
		}
		return err
	}
	return nil
}

// GetByLogin fetches a user by login. Returns domain.ErrUserNotFound if not found.
func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	user, err := WithRetry(ctx, 3, 100*time.Millisecond, func() (*domain.User, error) {
		var u domain.User
		err := r.db.QueryRowContext(ctx,
			`SELECT id, login, password_hash, created_at FROM users WHERE login = $1`,
			login,
		).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
		return &u, err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
