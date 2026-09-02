package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"betwin/auth-service/internal/client"
	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/database"
	"betwin/auth-service/internal/handler"
	"betwin/auth-service/internal/repository"
	"betwin/auth-service/internal/service"
	userv1 "betwin/pkg/proto/user/v1"
	walletv1 "betwin/pkg/proto/wallet/v1"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	//Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	//Connect to PostgreSQL pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database pool: %v", err)
	}
	defer pool.Close()
	log.Println("Successfully connected to PostgreSQL database!")

	userServiceConnection, err := grpc.NewClient(
		cfg.UserServiceGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to configure user service gRPC client: %v", err)
	}

	walletServiceConnection, err := grpc.NewClient(
		cfg.WalletServiceGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = userServiceConnection.Close()
		log.Fatalf("Failed to configure wallet service gRPC client: %v", err)
	}

	//Initialize layers (Dependency Injection)
	userRepo := repository.NewUserRepository(pool)
	profileProvisioner := client.NewUserProfileProvisioner(userv1.NewUserServiceClient(userServiceConnection))
	walletProvisioner := client.NewUserWalletProvisioner(walletv1.NewWalletServiceClient(walletServiceConnection))
	authSvc := service.NewAuthService(userRepo, profileProvisioner, walletProvisioner, cfg)
	authHandler := handler.NewAuthHandler(authSvc)

	router := chi.NewRouter()
	authHandler.RegisterRoutes(router)

	serverAddress := fmt.Sprintf(":%s", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:              serverAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Auth HTTP server is running on http://localhost%s", serverAddress)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("HTTP server stopped unexpectedly: %w", err)
		}
	}()

	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	select {
	case <-shutdownSignal.Done():
		log.Println("Shutdown signal received")
	case err := <-serverErrors:
		log.Printf("Server error: %v", err)
	}

	stopSignals()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownContext); err != nil {
		log.Printf("HTTP graceful shutdown failed: %v", err)
		if closeErr := httpServer.Close(); closeErr != nil {
			log.Printf("HTTP forced shutdown failed: %v", closeErr)
		}
	}

	if err := userServiceConnection.Close(); err != nil {
		log.Printf("Failed to close user-service gRPC connection: %v", err)
	}
	if err := walletServiceConnection.Close(); err != nil {
		log.Printf("Failed to close wallet-service gRPC connection: %v", err)
	}

	log.Println("Auth service stopped")
}
