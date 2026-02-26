// Package handlers implements HTTP request handlers.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/rs/zerolog"
)

type BalanceHandler struct {
	BalanceService *service.BalanceService
	Logger         zerolog.Logger
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.BalanceService.GetBalance(r.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID).Msg("get balance failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	})
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Order == "" || req.Sum <= 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err := h.BalanceService.WithdrawRequest(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientFunds):
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
			return
		case errors.Is(err, service.ErrInvalidOrderNumber):
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return
		default:
			h.Logger.Error().Err(err).Str("user_id", userID).Str("order", req.Order).Float64("sum", req.Sum).Msg("withdraw failed")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

type WithdrawalResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.BalanceService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID).Msg("get withdrawals failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	responses := make([]WithdrawalResponse, len(withdrawals))
	for i, w := range withdrawals {
		responses[i] = WithdrawalResponse{
			Order:       w.OrderNumber,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}
