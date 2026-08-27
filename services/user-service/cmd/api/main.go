package main

import (
	userv1 "betwin/pkg/proto/user/v1"
	user_gRPC "betwin/user-service/internal/API/grpc"
	"betwin/user-service/internal/API/rest/handler"
	"betwin/user-service/internal/config"
	"betwin/user-service/internal/database"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quic-go/quic-go/http3"
	grpc "google.golang.org/grpc"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(context, config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB pool: %v", err)
	}
	defer pool.Close()
	log.Println("Successfully connected to DB")

	profileRepo := repository.NewProfileRepository(pool)
	userService := service.NewUserService(profileRepo)

	grpcPort := ":50051"
	grpcListener, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC on %s: %v", grpcPort, err)
	}
	grpcServer := grpc.NewServer()
	userGrpcHandler := user_gRPC.NewUserGRPCHandler(userService)
	userv1.RegisterUserServiceServer(grpcServer, userGrpcHandler)

	go func() {
		log.Printf("gRPC server is running in parallel on tcp://localhost%s\n", grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server stopped unexpectedly: %v", err)
		}
	}()

	userHandler := handler.NewUserHandler(userService, config.JWTSecret)
	router := chi.NewRouter()

	serverAddress := fmt.Sprintf(":%s", config.Port)
	h3Server := &http3.Server{
		Addr:    serverAddress,
		Handler: router,
	}

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_ = h3Server.SetQUICHeaders(writer.Header())
			next.ServeHTTP(writer, request)
		})
	})

	userHandler.RegisterRoutes(router)
	log.Printf("User service running on HTTP/3 (QUIC/UDP) & HTTP/2 (TCP) at https://localhost%s\n", serverAddress)

	go func() {
		if err := h3Server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			log.Printf("HTTP/3 QUIC server error: %v\n", err)
		}
	}()

	if err := http.ListenAndServeTLS(serverAddress, "server.crt", "server.key", router); err != nil {
		log.Fatalf("HTTP/v2 server stopped unexpectedly: %v", err)
	}
}
