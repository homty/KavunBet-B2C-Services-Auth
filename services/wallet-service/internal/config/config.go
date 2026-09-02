package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort     string
	GRPCPort     string
	DatabaseURL  string
	JWTSecret    string
	AccessJWTTTL time.Duration
}

func Load() (*Config, error) {
	envPaths := []string{".env", "../.env", "../../.env", "../../../.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8083"
	}
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50052"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return &Config{
		HTTPPort:     httpPort,
		GRPCPort:     grpcPort,
		DatabaseURL:  databaseURL,
		JWTSecret:    jwtSecret,
		AccessJWTTTL: 15 * time.Minute,
	}, nil

}
