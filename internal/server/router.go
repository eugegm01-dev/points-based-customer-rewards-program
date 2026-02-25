// Package server provides HTTP server setup and route registration.
package server

import (
	"net/http"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/handlers"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/middleware"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/rs/zerolog"
)

// Dependencies holds services and repositories required by handlers.
type Dependencies struct {
	UserRepo    domain.UserRepository
	AuthService *service.AuthService
	AuthSecret  string
}

// NewRouter returns an HTTP handler with all routes registered.
func NewRouter(log zerolog.Logger, deps *Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(httplog.RequestLogger(log))

	r.Get("/health", handlers.Health)

	authHandler := &handlers.AuthHandler{
		AuthService: deps.AuthService,
		AuthSecret:  deps.AuthSecret,
		Logger:      log.With().Str("component", "auth").Logger(),
	}
	r.Post("/api/user/register", authHandler.Register)
	r.Post("/api/user/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.AuthSecret))
		// TODO: r.Post("/api/user/orders", ...)
		// TODO: r.Get("/api/user/orders", ...)
		// TODO: r.Get("/api/user/balance", ...)
		// TODO: r.Post("/api/user/balance/withdraw", ...)
		// TODO: r.Get("/api/user/withdrawals", ...)
	})

	return r
}
