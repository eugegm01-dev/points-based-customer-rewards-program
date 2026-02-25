// Package handlers implements HTTP request handlers.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings" // ✅ CORRECT PACKAGE FOR TrimSpace
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/rs/zerolog"
)

// OrderHandler holds dependencies for order endpoints.
type OrderHandler struct {
	OrderService *service.OrderService
	Logger       zerolog.Logger
}

// UploadOrder handles POST /api/user/orders.
func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// ✅ SAFE body reading with io.ReadAll
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Warn().Err(err).Msg("failed to read request body")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// ✅ CORRECT TrimSpace usage from strings package
	number := strings.TrimSpace(string(body))
	if number == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = h.OrderService.UploadOrder(r.Context(), userID, number)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateOrder):
			w.WriteHeader(http.StatusOK)
			return
		case errors.Is(err, service.ErrOrderBelongsToAnotherUser):
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
			return
		case errors.Is(err, service.ErrInvalidOrderNumber):
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return
		default:
			h.Logger.Error().Err(err).Str("user_id", userID).Str("order", number).Msg("upload order failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// OrderResponse is the JSON response for GET /api/user/orders.
type OrderResponse struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    *float64  `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// GetOrders handles GET /api/user/orders.
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.OrderService.GetOrdersByUserID(r.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID).Msg("get orders failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	responses := make([]OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = OrderResponse{
			Number:     order.Number,
			Status:     string(order.Status),
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		h.Logger.Warn().Err(err).Msg("failed to encode response")
	}
}

// BalanceResponse is the JSON response for GET /api/user/balance.
type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// GetBalance handles GET /api/user/balance.
func (h *OrderHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BalanceResponse{
		Current:   0.0,
		Withdrawn: 0.0,
	})
}
