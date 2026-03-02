package postgres_test

import (
	"context"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/repository/postgres"
)

func TestBalanceRepository_GetOrCreate(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	// Need a user first
	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "balanceuser", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	balanceRepo := postgres.NewBalanceRepository(db)

	// GetOrCreate should create a new balance
	b, err := balanceRepo.GetOrCreate(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	if b.UserID != user.ID {
		t.Errorf("expected userID %s, got %s", user.ID, b.UserID)
	}
	if b.Current != 0 || b.Withdrawn != 0 {
		t.Errorf("expected zero balance, got current=%.2f withdrawn=%.2f", b.Current, b.Withdrawn)
	}

	// Second call should return the same balance
	b2, err := balanceRepo.GetOrCreate(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b2.UserID != user.ID {
		t.Error("user ID mismatch")
	}
}

func TestBalanceRepository_Credit(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "credituser", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	balanceRepo := postgres.NewBalanceRepository(db)

	// Credit
	amount := 100.50
	if err := balanceRepo.Credit(context.Background(), user.ID, amount); err != nil {
		t.Fatalf("Credit failed: %v", err)
	}

	// Verify
	b, err := balanceRepo.GetOrCreate(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Current != amount {
		t.Errorf("expected current %.2f, got %.2f", amount, b.Current)
	}
	if b.Withdrawn != 0 {
		t.Errorf("expected withdrawn 0, got %.2f", b.Withdrawn)
	}
}

func TestBalanceRepository_Withdraw(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "withdrawuser", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	balanceRepo := postgres.NewBalanceRepository(db)

	// Add some funds
	if err := balanceRepo.Credit(context.Background(), user.ID, 200); err != nil {
		t.Fatal(err)
	}

	// Withdraw success
	orderNum := "12345678903" // valid Luhn
	sum := 50.25
	w, err := balanceRepo.Withdraw(context.Background(), user.ID, orderNum, sum)
	if err != nil {
		t.Fatalf("Withdraw failed: %v", err)
	}
	if w.OrderNumber != orderNum {
		t.Errorf("expected order %s, got %s", orderNum, w.OrderNumber)
	}
	if w.Sum != sum {
		t.Errorf("expected sum %.2f, got %.2f", sum, w.Sum)
	}

	// Check balance
	b, err := balanceRepo.GetOrCreate(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Current != 149.75 {
		t.Errorf("expected current 149.75, got %.2f", b.Current)
	}
	if b.Withdrawn != 50.25 {
		t.Errorf("expected withdrawn 50.25, got %.2f", b.Withdrawn)
	}

	// Insufficient funds
	_, err = balanceRepo.Withdraw(context.Background(), user.ID, orderNum, 200)
	if err != domain.ErrInsufficientFunds {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestBalanceRepository_GetWithdrawals(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "withdrawalsuser", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	balanceRepo := postgres.NewBalanceRepository(db)

	// Credit some funds
	if err := balanceRepo.Credit(context.Background(), user.ID, 1000); err != nil {
		t.Fatal(err)
	}

	// Make two withdrawals
	w1, err := balanceRepo.Withdraw(context.Background(), user.ID, "111", 100)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := balanceRepo.Withdraw(context.Background(), user.ID, "222", 200)
	if err != nil {
		t.Fatal(err)
	}

	withdrawals, err := balanceRepo.GetWithdrawals(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(withdrawals) != 2 {
		t.Fatalf("expected 2 withdrawals, got %d", len(withdrawals))
	}
	// Should be sorted by processed_at DESC, so w2 then w1
	if withdrawals[0].ID != w2.ID || withdrawals[1].ID != w1.ID {
		t.Error("withdrawals not sorted correctly")
	}
}
