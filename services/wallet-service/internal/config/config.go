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
	HTTPPort         string
	GRPCPort         string
	DatabaseURL      string
	JWTPublicKeyFile string
	JWTPublicKey     ed25519.PublicKey
	GRPCTLSCAFile    string
	GRPCTLSCertFile  string
	GRPCTLSKeyFile   string
	AccessJWTTTL     time.Duration
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

	databaseURL, err := requiredValue("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	jwtPublicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	grpcTLSCAFile := os.Getenv("GRPC_TLS_CA_FILE")
	grpcTLSCertFile := os.Getenv("GRPC_TLS_CERT_FILE")
	grpcTLSKeyFile := os.Getenv("GRPC_TLS_KEY_FILE")

	return &Config{
		HTTPPort:         httpPort,
		GRPCPort:         grpcPort,
		DatabaseURL:      databaseURL,
		JWTPublicKeyFile: jwtPublicKeyFile,
		GRPCTLSCAFile:    grpcTLSCAFile,
		GRPCTLSCertFile:  grpcTLSCertFile,
		GRPCTLSKeyFile:   grpcTLSKeyFile,
		AccessJWTTTL:     15 * time.Minute,
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

func LoadEd25519PublicKey(path string) (ed25519.PublicKey, error) {
	if path == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY_FILE environment variable is required")
	}

	encodedKey, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read JWT public key: %w", err)
	}
	block, _ := pem.Decode(encodedKey)
	if block == nil {
		return nil, fmt.Errorf("decode JWT public key PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse JWT public key: %w", err)
	}
	publicKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("JWT public key must be Ed25519")
	}
	return publicKey, nil
}
