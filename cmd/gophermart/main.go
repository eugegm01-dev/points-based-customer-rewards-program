package main

import (
	"context"
	"database/sql"
	"net/http"
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

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := migrate.Up(db); err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}
	log.Info().Msg("migrations applied")

	userRepo := postgres.NewUserRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	balanceRepo := postgres.NewBalanceRepository(db)

	authService := service.NewAuthService(userRepo)
	orderService := service.NewOrderService(orderRepo)
	balanceService := service.NewBalanceService(balanceRepo, orderRepo)

	accrualClient := accrual.NewClient(cfg.AccrualAddress)

	// Создаём пул воркеров (например, 5)
	workerPool := service.NewWorkerPool(5, orderService, balanceService, accrualClient, log)

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запускаем пул
	workerPool.Start(ctx)

	go func() {
		log.Info().Str("addr", cfg.RunAddress).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down...")

	// Останавливаем пул (он завершит все горутины по сигналу ctx.Done)
	workerPool.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server shutdown")
	}
	log.Info().Msg("shutdown complete")
}
