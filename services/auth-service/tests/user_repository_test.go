package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/database"
	"betwin/auth-service/internal/repository"
)

func setupTestDB(t *testing.T) repository.UserRepository {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	return repository.NewUserRepository(pool)
}

func TestCreateUserWithProfileAndWallet(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	uniqueUsername := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	passwordHash := "$2a$10$abcdefghijklmnopqrstuvwxyz123456" // Dummy bcrypt hash
	contactInfo := "test@example.com"

	user, err := repo.CreateUserWithProfileAndWallet(ctx, uniqueUsername, passwordHash, contactInfo)
	if err != nil {
		t.Fatalf("Expected successful user creation, got error: %v", err)
	}

	if user.ID <= 0 {
		t.Errorf("Expected valid user ID > 0, got %d", user.ID)
	}
	if user.Username != uniqueUsername {
		t.Errorf("Expected username %s, got %s", uniqueUsername, user.Username)
	}

	// 2. Duplicate username error
	_, err = repo.CreateUserWithProfileAndWallet(ctx, uniqueUsername, passwordHash, contactInfo)
	if !errors.Is(err, repository.ErrDuplicateUsername) {
		t.Errorf("Expected ErrDuplicateUsername on duplicate registration, got %v", err)
	}
}

func TestGetUserByUsername(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	uniqueUsername := fmt.Sprintf("finduser_%d", time.Now().UnixNano())
	passwordHash := "$2a$10$dummyhashval1234567890"

	createdUser, err := repo.CreateUserWithProfileAndWallet(ctx, uniqueUsername, passwordHash, "find@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	foundUser, err := repo.GetUserByUsername(ctx, uniqueUsername)
	if err != nil {
		t.Fatalf("Expected to find user by username, got error: %v", err)
	}
	if foundUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %d, got %d", createdUser.ID, foundUser.ID)
	}

	_, err = repo.GetUserByUsername(ctx, "non_existent_username_12345")
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	uniqueUsername := fmt.Sprintf("finduser_id_%d", time.Now().UnixNano())
	passwordHash := "$2a$10$dummyhashval1234567890"

	createdUser, err := repo.CreateUserWithProfileAndWallet(ctx, uniqueUsername, passwordHash, "find_id@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	foundUser, err := repo.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("Expected to find user by ID, got error: %v", err)
	}
	if foundUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %d, got %d", createdUser.ID, foundUser.ID)
	}
	if foundUser.Username != createdUser.Username {
		t.Errorf("Expected username %s, got %s", createdUser.Username, foundUser.Username)
	}

	_, err = repo.GetUserByID(ctx, 99999999)
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
