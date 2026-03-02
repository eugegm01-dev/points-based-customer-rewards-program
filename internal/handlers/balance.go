// Package handlers implements HTTP request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/validator"
	"github.com/rs/zerolog"
)

type BalanceService interface {
	GetBalance(ctx context.Context, userID string) (*domain.Balance, error)
	WithdrawRequest(ctx context.Context, userID, orderNumber string, sum float64) error
	GetWithdrawals(ctx context.Context, userID string) ([]*domain.Withdrawal, error)
}

type BalanceHandler struct {
	BalanceService BalanceService
	Logger         zerolog.Logger
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	balance, err := h.BalanceService.GetBalance(r.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID).Msg("get balance failed")
		WriteError(w, http.StatusInternalServerError, ErrInternalServer)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}); err != nil {
		h.Logger.Error().Err(err).Msg("failed to encode balance response")
	}
}

type WithdrawRequest struct {
	Order string  `json:"order" validate:"required"`
	Sum   float64 `json:"sum" validate:"required,gt=0"`
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}
	if errs := validator.ValidateStruct(req); errs != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		// ignore encode error
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "validation failed",
			"fields": errs,
		})
		return
	}

	if req.Order == "" || req.Sum <= 0 {
		WriteError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}
	err := h.BalanceService.WithdrawRequest(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientFunds):
			WriteError(w, http.StatusPaymentRequired, ErrInsufficientFunds)
			return
		case errors.Is(err, service.ErrInvalidOrderNumber):
			WriteError(w, http.StatusUnprocessableEntity, ErrInvalidOrderNumber)
			return
		default:
			h.Logger.Error().Err(err).Str("user_id", userID).Str("order", req.Order).Float64("sum", req.Sum).Msg("withdraw failed")
			WriteError(w, http.StatusInternalServerError, ErrInternalServer)
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
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	withdrawals, err := h.BalanceService.GetWithdrawals(r.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID).Msg("get withdrawals failed")
		WriteError(w, http.StatusInternalServerError, ErrInternalServer)
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
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		h.Logger.Error().Err(err).Msg("failed to encode withdrawals response")
	}
}
