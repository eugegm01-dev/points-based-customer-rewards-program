package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/handlers"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestOrderHandler_UploadOrder(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name       string
		userID     string
		body       string
		mockUpload func(ctx context.Context, userID, number string) error
		wantStatus int
	}{
		{
			name:       "success",
			userID:     "user123",
			body:       "12345678903",
			mockUpload: func(ctx context.Context, userID, number string) error { return nil },
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "duplicate own order",
			userID:     "user123",
			body:       "12345678903",
			mockUpload: func(ctx context.Context, userID, number string) error { return service.ErrDuplicateOrder },
			wantStatus: http.StatusOK,
		},
		{
			name:       "order belongs to another user",
			userID:     "user123",
			body:       "12345678903",
			mockUpload: func(ctx context.Context, userID, number string) error { return service.ErrOrderBelongsToAnotherUser },
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid number",
			userID:     "user123",
			body:       "123",
			mockUpload: func(ctx context.Context, userID, number string) error { return service.ErrInvalidOrderNumber },
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "unauthorized",
			userID:     "",
			body:       "12345678903",
			wantStatus: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockOrderService{uploadOrderFunc: tt.mockUpload}
			h := &handlers.OrderHandler{
				OrderService: mockSvc,
				Logger:       log,
			}

			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "text/plain")
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/orders", h.UploadOrder)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestOrderHandler_GetOrders(t *testing.T) {
	log := zerolog.Nop()
	tests := []struct {
		name          string
		userID        string
		mockGetOrders func(ctx context.Context, userID string) ([]*domain.Order, error)
		wantStatus    int
		wantLen       int
	}{
		{
			name:   "success with orders",
			userID: "user123",
			mockGetOrders: func(ctx context.Context, userID string) ([]*domain.Order, error) {
				return []*domain.Order{
					{Number: "111", Status: domain.OrderStatusProcessed, Accrual: float64Ptr(100)},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:   "no orders",
			userID: "user123",
			mockGetOrders: func(ctx context.Context, userID string) ([]*domain.Order, error) {
				return []*domain.Order{}, nil
			},
			wantStatus: http.StatusNoContent,
			wantLen:    0,
		},
		{
			name:   "service error",
			userID: "user123",
			mockGetOrders: func(ctx context.Context, userID string) ([]*domain.Order, error) {
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
			mockSvc := &mockOrderService{getOrdersByUserFunc: tt.mockGetOrders}
			h := &handlers.OrderHandler{
				OrderService: mockSvc,
				Logger:       log,
			}

			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Get("/orders", h.GetOrders)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantStatus == http.StatusOK {
				var resp []handlers.OrderResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp, tt.wantLen)
			}
		})
	}
}
func float64Ptr(v float64) *float64 { return &v }
