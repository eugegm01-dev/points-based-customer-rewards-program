package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/repository/postgres"
)

func TestOrderRepository_CreateAndGetByNumber(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "orderuser", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	orderRepo := postgres.NewOrderRepository(db)
	order := &domain.Order{
		UserID:     user.ID,
		Number:     "12345678903",
		Status:     domain.OrderStatusNew,
		UploadedAt: time.Now(),
	}

	// Create
	if err := orderRepo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if order.ID == 0 {
		t.Error("Create did not set ID")
	}

	// GetByNumber
	fetched, err := orderRepo.GetByNumber(context.Background(), "12345678903")
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if fetched.Number != order.Number {
		t.Errorf("expected number %s, got %s", order.Number, fetched.Number)
	}
	if fetched.UserID != user.ID {
		t.Errorf("expected userID %s, got %s", user.ID, fetched.UserID)
	}

	// Duplicate number should fail
	duplicate := &domain.Order{
		UserID:     user.ID,
		Number:     "12345678903",
		Status:     domain.OrderStatusNew,
		UploadedAt: time.Now(),
	}
	err = orderRepo.Create(context.Background(), duplicate)
	if err != domain.ErrDuplicateOrder {
		t.Errorf("expected ErrDuplicateOrder, got %v", err)
	}
}

func TestOrderRepository_GetByUserID(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user1 := &domain.User{Login: "user1", PasswordHash: "hash"}
	user2 := &domain.User{Login: "user2", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user1); err != nil {
		t.Fatal(err)
	}
	if err := userRepo.Create(context.Background(), user2); err != nil {
		t.Fatal(err)
	}

	orderRepo := postgres.NewOrderRepository(db)
	now := time.Now()
	orders := []*domain.Order{
		{UserID: user1.ID, Number: "111", Status: domain.OrderStatusNew, UploadedAt: now.Add(-2 * time.Hour)},
		{UserID: user1.ID, Number: "222", Status: domain.OrderStatusProcessing, UploadedAt: now.Add(-1 * time.Hour)},
		{UserID: user2.ID, Number: "333", Status: domain.OrderStatusProcessed, UploadedAt: now},
	}
	for _, o := range orders {
		if err := orderRepo.Create(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}

	// GetByUserID for user1
	user1Orders, err := orderRepo.GetByUserID(context.Background(), user1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(user1Orders) != 2 {
		t.Fatalf("expected 2 orders for user1, got %d", len(user1Orders))
	}
	// Should be sorted by uploaded_at DESC: "222" then "111"
	if user1Orders[0].Number != "222" || user1Orders[1].Number != "111" {
		t.Error("orders not sorted correctly")
	}

	// GetByUserID for user2
	user2Orders, err := orderRepo.GetByUserID(context.Background(), user2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(user2Orders) != 1 {
		t.Fatalf("expected 1 order for user2, got %d", len(user2Orders))
	}
	if user2Orders[0].Number != "333" {
		t.Errorf("expected order 333, got %s", user2Orders[0].Number)
	}
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "updatestatus", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	orderRepo := postgres.NewOrderRepository(db)
	order := &domain.Order{
		UserID:     user.ID,
		Number:     "444",
		Status:     domain.OrderStatusNew,
		UploadedAt: time.Now(),
	}
	if err := orderRepo.Create(context.Background(), order); err != nil {
		t.Fatal(err)
	}

	// Update to PROCESSING (no accrual)
	if err := orderRepo.UpdateStatus(context.Background(), order.ID, domain.OrderStatusProcessing, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := orderRepo.GetByNumber(context.Background(), "444")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.OrderStatusProcessing {
		t.Errorf("expected status PROCESSING, got %s", updated.Status)
	}
	if updated.ProcessedAt != nil {
		t.Error("ProcessedAt should be nil")
	}

	// Update to PROCESSED with accrual
	accrual := 123.45
	if err := orderRepo.UpdateStatus(context.Background(), order.ID, domain.OrderStatusProcessed, &accrual); err != nil {
		t.Fatal(err)
	}
	updated, err = orderRepo.GetByNumber(context.Background(), "444")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.OrderStatusProcessed {
		t.Errorf("expected status PROCESSED, got %s", updated.Status)
	}
	if updated.Accrual == nil || *updated.Accrual != accrual {
		t.Errorf("expected accrual %.2f, got %v", accrual, updated.Accrual)
	}
	if updated.ProcessedAt == nil {
		t.Error("ProcessedAt should be set")
	}
}

func TestOrderRepository_GetNewOrders(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "getnew", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	orderRepo := postgres.NewOrderRepository(db)
	orders := []*domain.Order{
		{UserID: user.ID, Number: "aaa", Status: domain.OrderStatusNew, UploadedAt: time.Now()},
		{UserID: user.ID, Number: "bbb", Status: domain.OrderStatusProcessing, UploadedAt: time.Now()},
		{UserID: user.ID, Number: "ccc", Status: domain.OrderStatusNew, UploadedAt: time.Now()},
	}
	for _, o := range orders {
		if err := orderRepo.Create(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}

	newOrders, err := orderRepo.GetNewOrders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(newOrders) != 2 {
		t.Fatalf("expected 2 new orders, got %d", len(newOrders))
	}
	// should include "aaa" and "ccc"
	numbers := map[string]bool{"aaa": true, "ccc": true}
	for _, o := range newOrders {
		if !numbers[o.Number] {
			t.Errorf("unexpected order %s in new orders", o.Number)
		}
	}
}

func TestOrderRepository_GetProcessingOrders(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	userRepo := postgres.NewUserRepository(db)
	user := &domain.User{Login: "getprocessing", PasswordHash: "hash"}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}

	orderRepo := postgres.NewOrderRepository(db)
	orders := []*domain.Order{
		{UserID: user.ID, Number: "ddd", Status: domain.OrderStatusNew, UploadedAt: time.Now()},
		{UserID: user.ID, Number: "eee", Status: domain.OrderStatusProcessing, UploadedAt: time.Now()},
		{UserID: user.ID, Number: "fff", Status: domain.OrderStatusProcessing, UploadedAt: time.Now()},
	}
	for _, o := range orders {
		if err := orderRepo.Create(context.Background(), o); err != nil {
			t.Fatal(err)
		}
	}

	processingOrders, err := orderRepo.GetProcessingOrders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(processingOrders) != 2 {
		t.Fatalf("expected 2 processing orders, got %d", len(processingOrders))
	}
	numbers := map[string]bool{"eee": true, "fff": true}
	for _, o := range processingOrders {
		if !numbers[o.Number] {
			t.Errorf("unexpected order %s in processing orders", o.Number)
		}
	}
}
