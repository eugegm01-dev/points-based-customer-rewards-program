package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/config"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/logger"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/migrate"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/repository/postgres"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/server"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const shutdownTimeout = 10 * time.Second

func main() {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	db, err := sql.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		log.Fatal().Err(err).Msg("open database")
	}
	defer db.Close()

	if err := migrate.Up(db); err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}
	log.Info().Msg("migrations applied")

	// ✅ CREATE ALL REPOSITORIES HERE (NOT in router!)
	userRepo := postgres.NewUserRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	balanceRepo := postgres.NewBalanceRepository(db) // ✅ ADDED

	// ✅ CREATE authService BEFORE Dependencies
	authService := service.NewAuthService(userRepo) // ✅ FIXED: was missing!

	deps := &server.Dependencies{
		UserRepo:    userRepo,
		OrderRepo:   orderRepo,
		BalanceRepo: balanceRepo, // ✅ ADDED
		AuthService: authService, // ✅ FIXED: was missing!
		AuthSecret:  cfg.AuthSecret,
	}

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: server.NewRouter(log, deps),
	}

	go func() {
		log.Info().Str("addr", cfg.RunAddress).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server shutdown")
	}
	log.Info().Msg("shutdown complete")
}
