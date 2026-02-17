package main

import (
	"context"
	"log"
	"net/http"
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
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create pg pool: %v", err)
	}
	defer pool.Close()

	var anchor service.MeasurementAnchor
	if cfg.BlockchainRPCURL != "" && cfg.BlockchainContractAddr != "" && cfg.BlockchainFromAddr != "" {
		a, err := blockchain.NewAnchor(cfg.BlockchainRPCURL, cfg.BlockchainContractAddr, cfg.BlockchainFromAddr)
		if err != nil {
			log.Printf("blockchain disabled: %v", err)
		} else {
			anchor = a
			log.Printf("blockchain anchor enabled: contract=%s", cfg.BlockchainContractAddr)
		}
	} else {
		log.Printf("blockchain anchor disabled: set BLOCKCHAIN_RPC_URL, BLOCKCHAIN_CONTRACT_ADDRESS, BLOCKCHAIN_FROM_ADDRESS")
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
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
