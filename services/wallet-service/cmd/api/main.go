package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	walletv1 "betwin/pkg/proto/wallet/v1"
	walletgrpc "betwin/wallet-service/internal/API/grpc/handler"
	walletrest "betwin/wallet-service/internal/API/rest/handler"
	"betwin/wallet-service/internal/config"
	"betwin/wallet-service/internal/database"
	"betwin/wallet-service/internal/repository"
	"betwin/wallet-service/internal/service"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	connectionContext, cancelConnection := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelConnection()

	pool, err := database.NewPostgresPool(connectionContext, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	walletRepository := repository.NewWalletRepository(pool)
	walletService := service.NewWalletService(walletRepository)
	walletHandler := walletgrpc.NewWalletGRPCHandler(walletService)

	address := fmt.Sprintf(":%s", cfg.GRPCPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC on %s: %v", address, err)
	}

	server := grpc.NewServer()
	walletv1.RegisterWalletServiceServer(server, walletHandler)

	walletHTTPHandler := walletrest.NewWalletHandler(
		walletService,
		cfg.JWTSecret,
	)

	router := chi.NewRouter()
	walletHTTPHandler.RegisterRoutes(router)

	serverAddress := fmt.Sprintf(":%s", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:              serverAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 2)
	go func() {
		log.Printf("Wallet gRPC service running on tcp://localhost%s", address)
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			serverErrors <- fmt.Errorf("wallet gRPC server: %w", err)
		}
	}()
	go func() {
		log.Printf("Wallet HTTP service running on http://localhost%s", serverAddress)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("wallet HTTP server: %w", err)
		}
	}()

	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	select {
	case <-shutdownSignal.Done():
		log.Println("Wallet service shutdown requested")
	case serveError := <-serverErrors:
		log.Printf("Wallet service stopping after server error: %v", serveError)
	}
	stopSignals()

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownContext); err != nil {
		log.Printf("Wallet HTTP graceful shutdown failed: %v", err)
		_ = httpServer.Close()
	}

	grpcStopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(grpcStopped)
	}()

	select {
	case <-grpcStopped:
		log.Println("Wallet gRPC service stopped gracefully")
	case <-shutdownContext.Done():
		log.Println("Wallet gRPC graceful shutdown timed out; forcing stop")
		server.Stop()
	}
}
