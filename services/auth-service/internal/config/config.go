package config

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort                    string
	DatabaseURL                 string
	JWTPrivateKeyFile           string
	JWTPrivateKey               ed25519.PrivateKey
	UserServiceGRPCAddress      string
	WalletServiceGRPCAddress    string
	GRPCTLSCAFile               string
	GRPCTLSCertFile             string
	GRPCTLSKeyFile              string
	UserServiceGRPCServerName   string
	WalletServiceGRPCServerName string
	AccessJWTTTL                time.Duration
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
		httpPort = "8081"
	}

	databaseURL, err := requiredValue("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	jwtPrivateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	userServiceGRPCAddress := os.Getenv("USER_SERVICE_GRPC_ADDRESS")
	if userServiceGRPCAddress == "" {
		userServiceGRPCAddress = "localhost:50051"
	}
	walletServiceGRPCAddress := os.Getenv("WALLET_SERVICE_GRPC_ADDRESS")
	if walletServiceGRPCAddress == "" {
		walletServiceGRPCAddress = "localhost:50052"
	}
	grpcTLSCAFile := os.Getenv("GRPC_TLS_CA_FILE")
	grpcTLSCertFile := os.Getenv("GRPC_TLS_CERT_FILE")
	grpcTLSKeyFile := os.Getenv("GRPC_TLS_KEY_FILE")
	userServiceGRPCServerName := os.Getenv("USER_SERVICE_GRPC_SERVER_NAME")
	if userServiceGRPCServerName == "" {
		userServiceGRPCServerName = "user-service"
	}
	walletServiceGRPCServerName := os.Getenv("WALLET_SERVICE_GRPC_SERVER_NAME")
	if walletServiceGRPCServerName == "" {
		walletServiceGRPCServerName = "wallet-service"
	}

	return &Config{
		HTTPPort:                    httpPort,
		DatabaseURL:                 databaseURL,
		JWTPrivateKeyFile:           jwtPrivateKeyFile,
		UserServiceGRPCAddress:      userServiceGRPCAddress,
		WalletServiceGRPCAddress:    walletServiceGRPCAddress,
		GRPCTLSCAFile:               grpcTLSCAFile,
		GRPCTLSCertFile:             grpcTLSCertFile,
		GRPCTLSKeyFile:              grpcTLSKeyFile,
		UserServiceGRPCServerName:   userServiceGRPCServerName,
		WalletServiceGRPCServerName: walletServiceGRPCServerName,
		AccessJWTTTL:                15 * time.Minute,
	}, nil

}

// requiredValue reads a setting directly or from a Compose/Docker secret file.
// Supplying both forms is rejected so the effective configuration is explicit.
func requiredValue(name string) (string, error) {
	value := os.Getenv(name)
	path := os.Getenv(name + "_FILE")
	if value != "" && path != "" {
		return "", fmt.Errorf("only one of %s or %s_FILE may be set", name, name)
	}
	if path != "" {
		contents, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", name, err)
		}
		value = strings.TrimSpace(string(contents))
	}
	if value == "" {
		return "", fmt.Errorf("%s or %s_FILE environment variable is required", name, name)
	}
	return value, nil
}

func LoadEd25519PrivateKey(path string) (ed25519.PrivateKey, error) {
	if path == "" {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY_FILE environment variable is required")
	}

	encodedKey, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read JWT private key: %w", err)
	}
	block, _ := pem.Decode(encodedKey)
	if block == nil {
		return nil, fmt.Errorf("decode JWT private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse JWT private key: %w", err)
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("JWT private key must be Ed25519")
	}
	return privateKey, nil
}
