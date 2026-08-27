package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"betwin/user-service/internal/config"
	"betwin/user-service/internal/database"
	"betwin/user-service/internal/domain"
	"betwin/user-service/internal/repository"
)

func setupTestDB(t *testing.T) (repository.ProfileRepository, int64) {
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

	// Create test user in users & user_profiles
	uniqueUsername := fmt.Sprintf("testuser_user_svc_%d", time.Now().UnixNano())
	var userID int64
	err = pool.QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id", uniqueUsername, "hash").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to insert test user into users table: %v", err)
	}

	repo := repository.NewProfileRepository(pool)
	if _, err := repo.ProvisionProfile(ctx, userID, uniqueUsername, "repo_test@example.com"); err != nil {
		t.Fatalf("Failed to provision test user profile: %v", err)
	}

	// Cleanup on test exit
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", userID)
		pool.Close()
	})

	return repo, userID
}

func TestProfileRepository_GetAndUpdate(t *testing.T) {
	repo, userID := setupTestDB(t)
	ctx := context.Background()

	// 1. Fetch profile
	profile, err := repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("Expected successful profile fetch, got error: %v", err)
	}
	if profile.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, profile.UserID)
	}

	// 2. Update profile
	displayName := "Master Predictor"
	bio := "Trading on crypto & sports prediction markets"
	avatar := "https://example.com/avatar_prod.png"

	updateReq := domain.UpdateProfileRequest{
		DisplayName: &displayName,
		Bio:         &bio,
		AvatarURL:   &avatar,
	}

	updated, err := repo.UpdateProfile(ctx, userID, updateReq)
	if err != nil {
		t.Fatalf("Expected successful profile update, got error: %v", err)
	}

	if updated.DisplayName == nil || *updated.DisplayName != "Master Predictor" {
		t.Errorf("Expected DisplayName 'Master Predictor', got %v", updated.DisplayName)
	}
	if updated.Bio == nil || *updated.Bio != bio {
		t.Errorf("Expected Bio %s, got %v", bio, updated.Bio)
	}

	// 3. Fetch public profile
	pub, err := repo.GetPublicProfileByID(ctx, userID)
	if err != nil {
		t.Fatalf("Expected successful public profile fetch, got error: %v", err)
	}
	if pub.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, pub.UserID)
	}
	if pub.Username == "" {
		t.Fatal("Expected public profile username to come from user_profiles")
	}
	if pub.DisplayName == nil || *pub.DisplayName != "Master Predictor" {
		t.Errorf("Expected DisplayName 'Master Predictor', got %v", pub.DisplayName)
	}
}

func TestProfileRepository_ProvisionIdempotent(t *testing.T) {
	repo, userID := setupTestDB(t)

	profile, err := repo.ProvisionProfile(context.Background(), userID, "ignored-retry-name", "changed@example.com")
	if err != nil {
		t.Fatalf("Expected repeated provisioning to succeed, got %v", err)
	}
	if profile.ContactInfo == nil || *profile.ContactInfo != "repo_test@example.com" {
		t.Fatalf("Repeated provisioning overwrote contact info: %v", profile.ContactInfo)
	}
}

func TestProfileRepository_NotFound(t *testing.T) {
	repo, _ := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetProfileByUserID(ctx, 99999999)
	if !errors.Is(err, repository.ErrProfileNotFound) {
		t.Errorf("Expected ErrProfileNotFound for non-existent ID, got %v", err)
	}
}
