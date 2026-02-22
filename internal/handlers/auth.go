package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/auth"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/rs/zerolog"
)

// AuthHandler holds dependencies for auth endpoints.
type AuthHandler struct {
	AuthService *service.AuthService
	AuthSecret  string
	Logger      zerolog.Logger
}

// RegisterRequest is the JSON body for POST /api/user/register and /api/user/login.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register handles POST /api/user/register. On success sets auth cookie and returns 200.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Login == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	u, err := h.AuthService.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateLogin) {
			http.Error(w, "login already taken", http.StatusConflict)
			return
		}
		h.Logger.Error().Err(err).Msg("register failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.CreateToken(h.AuthSecret, u.ID, u.Login)
	if err != nil {
		h.Logger.Error().Err(err).Msg("failed to create token")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	setAuthCookie(w, token)
	w.WriteHeader(http.StatusOK)
}

// Login handles POST /api/user/login. On success sets auth cookie and returns 200.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Login == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	u, err := h.AuthService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			http.Error(w, "invalid login or password", http.StatusUnauthorized)
			return
		}
		h.Logger.Error().Err(err).Msg("login failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.CreateToken(h.AuthSecret, u.ID, u.Login)
	if err != nil {
		h.Logger.Error().Err(err).Msg("failed to create token")
		http.Error(w, "internal error", http.StatusInternalServerError)
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
