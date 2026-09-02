package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"betwin/wallet-service/internal/domain"
	"betwin/wallet-service/internal/repository"
	"betwin/wallet-service/internal/service"
)

// MockWalletRepository simulates PostgreSQL wallet table in-memory.
type MockWalletRepository struct {
	wallets map[int64]*domain.UserWallet
}

func NewMockWalletRepository() *MockWalletRepository {
	return &MockWalletRepository{
		wallets: make(map[int64]*domain.UserWallet),
	}
}

func (m *MockWalletRepository) ProvisionWallet(ctx context.Context, userID int64) (*domain.UserWallet, error) {
	if existing, err := m.GetWalletByUserID(ctx, userID); err == nil {
		return existing, nil
	}
	wallet := &domain.UserWallet{
		ID:        int64(len(m.wallets) + 1),
		UserID:    userID,
		Balance:   1000,
		UpdatedAt: time.Now().UTC(),
	}
	m.wallets[wallet.ID] = wallet
	return wallet, nil
}

func (m *MockWalletRepository) DeleteWallet(_ context.Context, userID int64) error {
	for walletID, wallet := range m.wallets {
		if wallet.UserID == userID {
			delete(m.wallets, walletID)
		}
	}
	return nil
}

func (m *MockWalletRepository) GetWalletByUserID(ctx context.Context, userID int64) (*domain.UserWallet, error) {
	for _, w := range m.wallets {
		if w.UserID == userID {
			return w, nil
		}
	}
	return nil, repository.ErrWalletNotFound
}

func (m *MockWalletRepository) GetWalletByID(ctx context.Context, walletID int64) (*domain.UserWallet, error) {
	w, exists := m.wallets[walletID]
	if !exists {
		return nil, repository.ErrWalletNotFound
	}
	return w, nil
}

func (m *MockWalletRepository) ClaimDailyReward(ctx context.Context, userID int64, rewardAmount int64) (*domain.UserWallet, error) {
	w, err := m.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if w.LastDailyClaim != nil {
		if time.Since(*w.LastDailyClaim) < 24*time.Hour {
			return nil, repository.ErrDailyClaimAlreadyClaimed
		}
	}

	w.Balance += rewardAmount
	now := time.Now().UTC()
	w.LastDailyClaim = &now
	w.UpdatedAt = now

	return w, nil
}

func setupTestWalletService() (service.WalletService, *MockWalletRepository) {
	mockRepo := NewMockWalletRepository()
	svc := service.NewWalletService(mockRepo)
	return svc, mockRepo
}

func TestGetBalance_Success(t *testing.T) {
	svc, mockRepo := setupTestWalletService()
	ctx := context.Background()

	// Seed wallet
	mockRepo.wallets[10] = &domain.UserWallet{
		ID:        10,
		UserID:    1,
		Balance:   1000,
		UpdatedAt: time.Now(),
	}

	resp, err := svc.GetBalance(ctx, 1)
	if err != nil {
		t.Fatalf("Expected successful balance fetch, got: %v", err)
	}

	if resp.Balance != 1000 {
		t.Errorf("Expected balance 1000, got %d", resp.Balance)
	}
	if !resp.CanClaimDaily {
		t.Errorf("Expected CanClaimDaily to be true for new user, got false")
	}
}

func TestClaimDaily_Success(t *testing.T) {
	svc, mockRepo := setupTestWalletService()
	ctx := context.Background()

	mockRepo.wallets[20] = &domain.UserWallet{
		ID:        20,
		UserID:    2,
		Balance:   1000,
		UpdatedAt: time.Now(),
	}

	claimResp, err := svc.ClaimDaily(ctx, 2)
	if err != nil {
		t.Fatalf("Expected successful daily claim, got: %v", err)
	}

	if claimResp.CoinsClaimed != 100 {
		t.Errorf("Expected CoinsClaimed 100, got %d", claimResp.CoinsClaimed)
	}
	if claimResp.NewBalance != 1100 {
		t.Errorf("Expected NewBalance 1100, got %d", claimResp.NewBalance)
	}

	// 2nd claim immediately should fail
	_, err = svc.ClaimDaily(ctx, 2)
	if !errors.Is(err, repository.ErrDailyClaimAlreadyClaimed) {
		t.Errorf("Expected ErrDailyClaimAlreadyClaimed on 2nd claim, got %v", err)
	}
}

func TestProvisionWallet_Idempotent(t *testing.T) {
	svc, mockRepo := setupTestWalletService()
	ctx := context.Background()

	first, err := svc.ProvisionWallet(ctx, 52)
	if err != nil {
		t.Fatalf("Expected first provisioning to succeed, got %v", err)
	}
	second, err := svc.ProvisionWallet(ctx, 52)
	if err != nil {
		t.Fatalf("Expected repeated provisioning to succeed, got %v", err)
	}

	if first != second {
		t.Fatal("Expected repeated provisioning to return the existing wallet")
	}
	if len(mockRepo.wallets) != 1 {
		t.Fatalf("Expected one wallet, got %d", len(mockRepo.wallets))
	}
	if second.Balance != 1000 {
		t.Fatalf("Expected starting balance to remain 1000, got %d", second.Balance)
	}
}

func TestDeleteWallet_Idempotent(t *testing.T) {
	svc, mockRepo := setupTestWalletService()
	if _, err := svc.ProvisionWallet(context.Background(), 52); err != nil {
		t.Fatalf("provision wallet: %v", err)
	}

	if err := svc.DeleteWallet(context.Background(), 52); err != nil {
		t.Fatalf("first deletion failed: %v", err)
	}
	if err := svc.DeleteWallet(context.Background(), 52); err != nil {
		t.Fatalf("repeated deletion failed: %v", err)
	}
	if _, err := mockRepo.GetWalletByUserID(context.Background(), 52); !errors.Is(err, repository.ErrWalletNotFound) {
		t.Fatalf("expected wallet not found after deletion, got %v", err)
	}
}
