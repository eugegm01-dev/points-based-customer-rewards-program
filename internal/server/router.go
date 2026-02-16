// Package server provides HTTP server setup and route registration.
package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/handlers"
	"github.com/rs/zerolog"
)

// NewRouter returns an HTTP handler with all routes registered.
func NewRouter(log zerolog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(httplog.RequestLogger(log))

	r.Get("/health", handlers.Health)

	// TODO: r.Route("/api/user", ...) — auth, orders, balance
	return r
}
