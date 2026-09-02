package main

import (
	userv1 "betwin/pkg/proto/user/v1"
	usergrpc "betwin/user-service/internal/API/grpc"
	userrest "betwin/user-service/internal/API/rest/handler"
	"betwin/user-service/internal/config"
	"betwin/user-service/internal/database"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"
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

	"github.com/go-chi/chi/v5"
	grpc "google.golang.org/grpc"
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
		log.Fatalf("Failed to connect to DB pool: %v", err)
	}
	defer pool.Close()
	log.Println("Successfully connected to DB")

	profileRepo := repository.NewProfileRepository(pool)
	userService := service.NewUserService(profileRepo)

	grpcAddress := ":" + cfg.GRPCPort
	grpcListener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC on %s: %v", grpcAddress, err)
	}
	defer grpcListener.Close()

	grpcServer := grpc.NewServer()
	userGrpcHandler := usergrpc.NewUserGRPCHandler(userService)
	userv1.RegisterUserServiceServer(grpcServer, userGrpcHandler)

	userHTTPHandler := userrest.NewUserHandler(userService, cfg.JWTSecret)

	router := chi.NewRouter()
	userHTTPHandler.RegisterRoutes(router)

	httpAddress := fmt.Sprintf(":%s", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:              httpAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 2)

	go func() {
		log.Printf("gRPC server is running on tcp://localhost%s", grpcAddress)
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			serverErrors <- fmt.Errorf("gRPC server stopped unexpectedly: %w", err)
		}
	}()

	go func() {
		log.Printf("HTTP server is running on http://localhost%s", httpAddress)
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

	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()

	select {
	case <-grpcStopped:
		log.Println("gRPC server stopped gracefully")
	case <-shutdownContext.Done():
		log.Println("gRPC graceful shutdown timed out; forcing stop")
		grpcServer.Stop()
		<-grpcStopped
	}

	log.Println("User service stopped")
}
