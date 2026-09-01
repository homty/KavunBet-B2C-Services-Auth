package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"betwin/auth-service/internal/client"
	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/database"
	"betwin/auth-service/internal/handler"
	"betwin/auth-service/internal/repository"
	"betwin/auth-service/internal/service"
	userv1 "betwin/pkg/proto/user/v1"

	"github.com/go-chi/chi/v5"
	"github.com/quic-go/quic-go/http3"
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
	defer userServiceConnection.Close()

	//Initialize layers (Dependency Injection)
	userRepo := repository.NewUserRepository(pool)
	profileProvisioner := client.NewUserProfileProvisioner(userv1.NewUserServiceClient(userServiceConnection))
	authSvc := service.NewAuthService(userRepo, profileProvisioner, cfg)
	authHandler := handler.NewAuthHandler(authSvc)

	//Router & HTTP/3 Alt-Svc middleware
	r := chi.NewRouter()

	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	h3Server := &http3.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	//Alt-Svc header notifies clients that HTTP/3 over QUIC/UDP is available on port
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			_ = h3Server.SetQUICHeaders(w.Header())
			next.ServeHTTP(w, req)
		})
	})

	authHandler.RegisterRoutes(r)

	log.Printf("Auth Service running on HTTP/3 (QUIC/UDP) & HTTP/2 (TCP) at https://localhost%s\n", serverAddr)

	//HTTP/3 Server over UDP in background goroutine
	go func() {
		if err := h3Server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			log.Printf("HTTP/3 QUIC server error: %v\n", err)
		}
	}()

	//HTTP/2 & HTTP/1.1 fallback server over TCP
	if err := http.ListenAndServeTLS(serverAddr, "server.crt", "server.key", r); err != nil {
		log.Fatalf("HTTP/2 server stopped unexpectedly: %v", err)
	}
}
