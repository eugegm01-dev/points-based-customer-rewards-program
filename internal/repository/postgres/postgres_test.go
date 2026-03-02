package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/migrate" // add this

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations
	if err := migrate.Up(db); err != nil {
		t.Fatal(err)
	}

	return db, func() {
		db.Close()
		pgContainer.Terminate(ctx)
	}
}
