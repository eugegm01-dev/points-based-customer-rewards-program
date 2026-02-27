package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/accrual"
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

	// ✅ CREATE ALL REPOSITORIES HERE
	userRepo := postgres.NewUserRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	balanceRepo := postgres.NewBalanceRepository(db)

	// ✅ CREATE SERVICES
	authService := service.NewAuthService(userRepo)
	orderService := service.NewOrderService(orderRepo)
	balanceService := service.NewBalanceService(balanceRepo, orderRepo)

	// ✅ CREATE ACCRUAL CLIENT & WORKER
	accrualClient := accrual.NewClient(cfg.AccrualAddress)
	accrualWorker := service.NewAccrualWorker(orderService, balanceService, accrualClient, log, 10*time.Second)

	deps := &server.Dependencies{
		UserRepo:    userRepo,
		OrderRepo:   orderRepo,
		BalanceRepo: balanceRepo,
		AuthService: authService,
		AuthSecret:  cfg.AuthSecret,
	}

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: server.NewRouter(log, deps),
	}

	// ✅ START WORKER
	ctx, cancel := context.WithCancel(context.Background())
	go accrualWorker.Run(ctx)

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

	// ✅ STOP WORKER
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error().Err(err).Msg("server shutdown")
	}
	log.Info().Msg("shutdown complete")
}
