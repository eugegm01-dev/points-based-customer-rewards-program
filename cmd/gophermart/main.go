package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/config"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/logger"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/server"
)

const shutdownTimeout = 10 * time.Second

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: server.NewRouter(log),
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
	// TODO: close DB when added
	log.Info().Msg("shutdown complete")
}
