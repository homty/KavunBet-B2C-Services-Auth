package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"betwin/pkg/config"
	"betwin/pkg/database"
	"betwin/wallet-service/internal/repository"
)

func setupTestWalletDB(t *testing.T, createWallet bool) (repository.WalletRepository, int64) {
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

	// Create test user in users & user_wallets
	uniqueUsername := fmt.Sprintf("testuser_wallet_%d", time.Now().UnixNano())
	var userID int64
	err = pool.QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id", uniqueUsername, "hash").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to insert test user into users table: %v", err)
	}

	if createWallet {
		_, err = pool.Exec(ctx, "INSERT INTO user_wallets (user_id, coin_balance) VALUES ($1, 1000)", userID)
		if err != nil {
			t.Fatalf("Failed to insert test user_wallet: %v", err)
		}
	}

	// Cleanup on test exit
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM users WHERE id = $1", userID)
		pool.Close()
	})

	return repository.NewWalletRepository(pool), userID
}

func TestWalletRepository_GetAndClaim(t *testing.T) {
	repo, userID := setupTestWalletDB(t, true)
	ctx := context.Background()

	// 1. Fetch wallet by user ID
	wallet, err := repo.GetWalletByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("Expected successful wallet fetch, got error: %v", err)
	}
	if wallet.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, wallet.UserID)
	}
	if wallet.Balance != 1000 {
		t.Errorf("Expected initial balance 1000, got %d", wallet.Balance)
	}

	// 2. Claim daily reward (+100 coins)
	updated, err := repo.ClaimDailyReward(ctx, userID, 100)
	if err != nil {
		t.Fatalf("Expected successful daily claim, got error: %v", err)
	}
	if updated.Balance != 1100 {
		t.Errorf("Expected updated balance 1100, got %d", updated.Balance)
	}

	// 3. Second claim within 24 hours should fail
	_, err = repo.ClaimDailyReward(ctx, userID, 100)
	if !errors.Is(err, repository.ErrDailyClaimAlreadyClaimed) {
		t.Errorf("Expected ErrDailyClaimAlreadyClaimed, got %v", err)
	}

	// 4. Fetch wallet by Wallet ID
	walletByID, err := repo.GetWalletByID(ctx, wallet.ID)
	if err != nil {
		t.Fatalf("Expected successful wallet fetch by ID, got error: %v", err)
	}
	if walletByID.ID != wallet.ID {
		t.Errorf("Expected Wallet ID %d, got %d", wallet.ID, walletByID.ID)
	}
}

func TestWalletRepository_NotFound(t *testing.T) {
	repo, _ := setupTestWalletDB(t, true)
	ctx := context.Background()

	_, err := repo.GetWalletByUserID(ctx, 99999999)
	if !errors.Is(err, repository.ErrWalletNotFound) {
		t.Errorf("Expected ErrWalletNotFound for non-existent ID, got %v", err)
	}
}

func TestWalletRepository_ProvisionIdempotent(t *testing.T) {
	repo, userID := setupTestWalletDB(t, false)
	ctx := context.Background()

	first, err := repo.ProvisionWallet(ctx, userID)
	if err != nil {
		t.Fatalf("Expected first provisioning to succeed, got %v", err)
	}
	second, err := repo.ProvisionWallet(ctx, userID)
	if err != nil {
		t.Fatalf("Expected repeated provisioning to succeed, got %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("Expected the same wallet ID, got %d and %d", first.ID, second.ID)
	}
	if second.Balance != 1000 {
		t.Fatalf("Expected starting balance to remain 1000, got %d", second.Balance)
	}
}
