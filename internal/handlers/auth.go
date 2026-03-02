package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/auth"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/validator"
	"github.com/rs/zerolog"
)

// AuthHandler holds dependencies for auth endpoints.
type AuthService interface {
	Register(ctx context.Context, login, password string) (*domain.User, error)
	Login(ctx context.Context, login, password string) (*domain.User, error)
}

type AuthHandler struct {
	AuthService AuthService
	AuthSecret  string
	Logger      zerolog.Logger
}

// RegisterRequest is the JSON body for POST /api/user/register and /api/user/login.
type RegisterRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// Register handles POST /api/user/register. On success sets auth cookie and returns 200.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}
	if errs := validator.ValidateStruct(req); errs != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "validation failed",
			"fields": errs,
		})
		return
	}

	if req.Login == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}
	u, err := h.AuthService.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateLogin) {
			WriteError(w, http.StatusConflict, ErrLoginAlreadyTaken)
			return
		}
		h.Logger.Error().Err(err).Msg("register failed")
		WriteError(w, http.StatusInternalServerError, ErrInternalServer)
		return
	}
	token, err := auth.CreateToken(h.AuthSecret, u.ID, u.Login)
	if err != nil {
		h.Logger.Error().Err(err).Msg("failed to create token")
		WriteError(w, http.StatusInternalServerError, ErrInternalServer)
		return
	}
	setAuthCookie(w, token)
	w.WriteHeader(http.StatusOK)
}

// Login handles POST /api/user/login. On success sets auth cookie and returns 200.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrBadRequest)
		return
	}
	if errs := validator.ValidateStruct(req); errs != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "validation failed",
			"fields": errs,
		})
		return
	}

	u, err := h.AuthService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			WriteError(w, http.StatusUnauthorized, ErrInvalidCredentials)
			return
		}
		h.Logger.Error().Err(err).Msg("login failed")
		WriteError(w, http.StatusInternalServerError, ErrInternalServer)
		return
	}
	token, err := auth.CreateToken(h.AuthSecret, u.ID, u.Login)
	if err != nil {
		h.Logger.Error().Err(err).Msg("failed to create token")
		WriteError(w, http.StatusInternalServerError, ErrInternalServer)
		return
	}
	setAuthCookie(w, token)
	w.WriteHeader(http.StatusOK)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24h in seconds
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set true when served over HTTPS
	})
}
