package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	JWTSecret              string
	UserServiceGRPCAddress string
	AccessJWTTTL           time.Duration
}

func Load() (*Config, error) {
	envPaths := []string{".env", "../.env", "../../.env", "../../../.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	userServiceGRPCAddress := os.Getenv("USER_SERVICE_GRPC_ADDRESS")
	if userServiceGRPCAddress == "" {
		userServiceGRPCAddress = "localhost:50051"
	}

	return &Config{
		Port:                   port,
		DatabaseURL:            databaseURL,
		JWTSecret:              jwtSecret,
		UserServiceGRPCAddress: userServiceGRPCAddress,
		AccessJWTTTL:           15 * time.Minute,
	}, nil

}
