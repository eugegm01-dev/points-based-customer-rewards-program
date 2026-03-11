package postgres

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func WithRetry[T any](ctx context.Context, maxAttempts int, baseDelay time.Duration,
	fn func() (T, error)) (T, error) {

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		if !isTransientDBError(err) {
			var zero T
			return zero, err // не ретраим постоянные ошибки
		}
		lastErr = err
		delay := baseDelay * time.Duration(1<<attempt)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		}
	}
	var zero T
	return zero, lastErr
}

// Retry выполняет функцию fn с повторными попытками при временных ошибках.
// Возвращает только ошибку (если все попытки исчерпаны).
func Retry(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error {
	_, err := WithRetry(ctx, maxAttempts, baseDelay, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

func isTransientDBError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Коды, которые можно повторить:
		// 40001 - serialization_failure
		// 40P01 - deadlock_detected
		// все 08XXX - connection errors
		if pgErr.Code == "40001" || pgErr.Code == "40P01" {
			return true
		}
		if len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
			return true
		}
		return false
	}

	// Сетевые ошибки с таймаутом
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Контекстные ошибки (но они обрабатываются отдельно в цикле)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// Некоторые специфические строки
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer")
}
