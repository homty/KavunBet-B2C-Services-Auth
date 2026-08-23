package main

import (
	"betwin/user-service/internal/API/handler"
	"betwin/user-service/internal/config"
	"betwin/user-service/internal/database"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quic-go/quic-go/http3"
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
