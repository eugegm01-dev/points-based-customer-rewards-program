package postgres_test

import (
	"context"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/repository/postgres"
)

func TestUserRepository_CreateAndGetByLogin(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	repo := postgres.NewUserRepository(db)
	user := &domain.User{
		Login:        "testuser",
		PasswordHash: "hashedpassword",
	}

	// Create
	err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.ID == "" {
		t.Error("Create did not set ID")
	}

	// GetByLogin
	fetched, err := repo.GetByLogin(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetByLogin failed: %v", err)
	}
	if fetched.Login != "testuser" {
		t.Errorf("expected login testuser, got %s", fetched.Login)
	}

	// Duplicate login should fail
	duplicate := &domain.User{Login: "testuser", PasswordHash: "another"}
	err = repo.Create(context.Background(), duplicate)
	if err != domain.ErrDuplicateLogin {
		t.Errorf("expected ErrDuplicateLogin, got %v", err)
	}

	// Get non-existent
	_, err = repo.GetByLogin(context.Background(), "nosuchuser")
	if err != domain.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
