package service

import (
	"context"
	"fmt"
	"time"

	"betwin/wallet-service/internal/domain"
	"betwin/wallet-service/internal/repository"
)

type WalletService interface {
	GetBalance(ctx context.Context, userID int64) (*domain.WalletBalanceResponse, error)
	ClaimDaily(ctx context.Context, userID int64) (*domain.DailyClaimResponse, error)
	GetWalletByID(ctx context.Context, walletID int64) (*domain.UserWallet, error)
}

type walletService struct {
	walletRepository repository.WalletRepository
}

func NewWalletService(walletRepository repository.WalletRepository) WalletService {
	return &walletService{
		walletRepository: walletRepository,
	}
}

const rewardAmount int64 = 100

// GetBalance retrieves wallet balance and calculates if daily reward is claimable.
func (s *walletService) GetBalance(ctx context.Context, userID int64) (*domain.WalletBalanceResponse, error) {
	wallet, err := s.walletRepository.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	canClaim := true
	if wallet.LastDailyClaim != nil {
		if time.Since(*wallet.LastDailyClaim) < 24*time.Hour {
			canClaim = false
		}
	}

	return &domain.WalletBalanceResponse{
		WalletID:       wallet.ID,
		UserID:         wallet.UserID,
		Balance:        wallet.Balance,
		LastDailyClaim: wallet.LastDailyClaim,
		CanClaimDaily:  canClaim,
	}, nil
}

// ClaimDaily grants +100 daily reward coins to the user.
func (s *walletService) ClaimDaily(ctx context.Context, userID int64) (*domain.DailyClaimResponse, error) {
	updatedWallet, err := s.walletRepository.ClaimDailyReward(ctx, userID, rewardAmount)
	if err != nil {
		return nil, err
	}

	var nextAvailable time.Time
	if updatedWallet.LastDailyClaim != nil {
		nextAvailable = updatedWallet.LastDailyClaim.Add(24 * time.Hour).UTC()
	}

	return &domain.DailyClaimResponse{
		Message:              fmt.Sprintf("Successfully claimed +%d daily bonus coins!", rewardAmount),
		WalletID:             updatedWallet.ID,
		CoinsClaimed:         rewardAmount,
		NewBalance:           updatedWallet.Balance,
		NextClaimAvailableAt: nextAvailable,
	}, nil
}

// GetWalletByID retrieves wallet metadata by wallet_id.
func (s *walletService) GetWalletByID(ctx context.Context, walletID int64) (*domain.UserWallet, error) {
	return s.walletRepository.GetWalletByID(ctx, walletID)
}
