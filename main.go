package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eco-api/internal/blockchain"
	"eco-api/internal/config"
	"eco-api/internal/handler"
	"eco-api/internal/repository"
	"eco-api/internal/router"
	"eco-api/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("create pg pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	var anchor service.MeasurementAnchor
	if cfg.BlockchainRPCURL != "" {
		// config.validate() guarantees all three vars are present and valid.
		a, err := blockchain.NewAnchor(cfg.BlockchainRPCURL, cfg.BlockchainContractAddr, cfg.BlockchainFromAddr)
		if err != nil {
			slog.Error("blockchain init", "error", err)
			os.Exit(1)
		}
		anchor = a
		slog.Info("blockchain anchor enabled", "contract", cfg.BlockchainContractAddr)
	} else {
		slog.Info("blockchain anchor disabled")
	}

	measurementRepo := repository.NewMeasurementRepository(pool)
	measurementService := service.NewMeasurementService(measurementRepo, anchor)
	h := handler.New(measurementService, pool)
	engine := router.New(h)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown", "error", err)
		}
	}()

	slog.Info("server starting", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("listen and serve", "error", err)
		os.Exit(1)
	}
}
