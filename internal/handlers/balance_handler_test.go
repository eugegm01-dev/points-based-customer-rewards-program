package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/handlers"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestBalanceHandler_GetBalance(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name           string
		userID         string // set in context
		mockGetBalance func(ctx context.Context, userID string) (*domain.Balance, error)
		wantStatus     int
		wantBody       map[string]float64
	}{
		{
			name:   "success",
			userID: "user123",
			mockGetBalance: func(ctx context.Context, userID string) (*domain.Balance, error) {
				return &domain.Balance{Current: 100.5, Withdrawn: 20}, nil
			},
			wantStatus: http.StatusOK,
			wantBody:   map[string]float64{"current": 100.5, "withdrawn": 20},
		},
		{
			name:       "unauthorized",
			userID:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: "user123",
			mockGetBalance: func(ctx context.Context, userID string) (*domain.Balance, error) {
				return nil, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockBalanceService{getBalanceFunc: tt.mockGetBalance}
			h := &handlers.BalanceHandler{
				BalanceService: mockSvc,
				Logger:         log,
			}

			req := httptest.NewRequest(http.MethodGet, "/balance", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/balance", h.GetBalance)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusOK {
				var resp map[string]float64
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.InDelta(t, tt.wantBody["current"], resp["current"], 0.01)
				assert.InDelta(t, tt.wantBody["withdrawn"], resp["withdrawn"], 0.01)
			}
		})
	}
}

func TestBalanceHandler_Withdraw(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name         string
		userID       string
		reqBody      interface{}
		mockWithdraw func(ctx context.Context, userID, orderNumber string, sum float64) error
		wantStatus   int
	}{
		{
			name:    "success",
			userID:  "user123",
			reqBody: map[string]interface{}{"order": "12345678903", "sum": 50},
			mockWithdraw: func(ctx context.Context, userID, orderNumber string, sum float64) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized",
			userID:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "invalid order number",
			userID:  "user123",
			reqBody: map[string]interface{}{"order": "123", "sum": 50},
			mockWithdraw: func(ctx context.Context, userID, orderNumber string, sum float64) error {
				return service.ErrInvalidOrderNumber
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:    "insufficient funds",
			userID:  "user123",
			reqBody: map[string]interface{}{"order": "12345678903", "sum": 1000},
			mockWithdraw: func(ctx context.Context, userID, orderNumber string, sum float64) error {
				return domain.ErrInsufficientFunds
			},
			wantStatus: http.StatusPaymentRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockBalanceService{withdrawRequestFunc: tt.mockWithdraw}
			h := &handlers.BalanceHandler{
				BalanceService: mockSvc,
				Logger:         log,
			}

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/withdraw", h.Withdraw)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestBalanceHandler_GetWithdrawals(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name               string
		userID             string
		mockGetWithdrawals func(ctx context.Context, userID string) ([]*domain.Withdrawal, error)
		wantStatus         int
		wantLen            int
	}{
		{
			name:   "success with withdrawals",
			userID: "user123",
			mockGetWithdrawals: func(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
				return []*domain.Withdrawal{
					{OrderNumber: "111", Sum: 50, ProcessedAt: time.Now()},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:   "no withdrawals",
			userID: "user123",
			mockGetWithdrawals: func(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
				return []*domain.Withdrawal{}, nil
			},
			wantStatus: http.StatusNoContent,
			wantLen:    0,
		},
		{
			name:   "service error",
			userID: "user123",
			mockGetWithdrawals: func(ctx context.Context, userID string) ([]*domain.Withdrawal, error) {
				return nil, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "unauthorized",
			userID:     "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockBalanceService{getWithdrawalsFunc: tt.mockGetWithdrawals}
			h := &handlers.BalanceHandler{
				BalanceService: mockSvc,
				Logger:         log,
			}

			req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/withdrawals", h.GetWithdrawals)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusOK {
				var resp []handlers.WithdrawalResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp, tt.wantLen)
			}
		})
	}
}
