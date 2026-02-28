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

type Dependencies struct {
	UserRepo    domain.UserRepository
	OrderRepo   domain.OrderRepository
	BalanceRepo domain.BalanceRepository // ✅ ADDED
	AuthService *service.AuthService
	AuthSecret  string
}

func NewRouter(log zerolog.Logger, deps *Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(httplog.RequestLogger(log))
	r.Use(middleware.Gzip) // ✅ ADD GZIP MIDDLEWARE

	r.Get("/health", handlers.Health)

	// Public routes
	authHandler := &handlers.AuthHandler{
		AuthService: deps.AuthService,
		AuthSecret:  deps.AuthSecret,
		Logger:      log.With().Str("component", "auth").Logger(),
	}
	r.Post("/api/user/register", authHandler.Register)
	r.Post("/api/user/login", authHandler.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(deps.AuthSecret))

		// Order handlers
		orderHandler := &handlers.OrderHandler{
			OrderService: service.NewOrderService(deps.OrderRepo),
			Logger:       log.With().Str("component", "orders").Logger(),
		}
		r.Post("/api/user/orders", orderHandler.UploadOrder)
		r.Get("/api/user/orders", orderHandler.GetOrders)

		// Balance handlers ✅ CORRECT WIRING (no db here!)
		balanceHandler := &handlers.BalanceHandler{
			BalanceService: service.NewBalanceService(deps.BalanceRepo, deps.OrderRepo),
			Logger:         log.With().Str("component", "balance").Logger(),
		}
		r.Get("/api/user/balance", balanceHandler.GetBalance)
		r.Post("/api/user/balance/withdraw", balanceHandler.Withdraw)
		r.Get("/api/user/withdrawals", balanceHandler.GetWithdrawals)
	})

	return r
}
