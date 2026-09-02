package repository

import (
	"betwin/wallet-service/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWalletNotFound           = errors.New("Wallet not found")
	ErrDailyClaimAlreadyClaimed = errors.New("Daily bonus already claimed within 24 hours")
)

type WalletRepository interface {
	ProvisionWallet(context context.Context, userID int64) (*domain.UserWallet, error)
	DeleteWallet(context context.Context, userID int64) error
	GetWalletByUserID(context context.Context, userID int64) (*domain.UserWallet, error)
	GetWalletByID(context context.Context, walletID int64) (*domain.UserWallet, error)
	ClaimDailyReward(context context.Context, userID int64, rewardAmount int64) (*domain.UserWallet, error)
}

func (repository *postgresWalletRepository) DeleteWallet(context context.Context, userID int64) error {
	if _, err := repository.pool.Exec(context, `DELETE FROM wallet_service.wallets WHERE user_id = $1;`, userID); err != nil {
		return fmt.Errorf("failed to delete wallet: %w", err)
	}
	return nil
}

func (repository *postgresWalletRepository) ProvisionWallet(context context.Context, userID int64) (*domain.UserWallet, error) {
	query := `
		INSERT INTO wallet_service.wallets (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING;
	`
	if _, err := repository.pool.Exec(context, query, userID); err != nil {
		return nil, fmt.Errorf("failed to provision wallet: %w", err)
	}

	return repository.GetWalletByUserID(context, userID)
}

type postgresWalletRepository struct {
	pool *pgxpool.Pool
}

func NewWalletRepository(pool *pgxpool.Pool) WalletRepository {
	return &postgresWalletRepository{pool: pool}
}

func (repository *postgresWalletRepository) GetWalletByUserID(context context.Context, userID int64) (*domain.UserWallet, error) {
	query := `
		SELECT id, user_id, coin_balance, last_daily_claim, updated_at
		FROM wallet_service.wallets
		WHERE user_id = $1;
	`

	var wallet domain.UserWallet
	err := repository.pool.QueryRow(context, query, userID).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.LastDailyClaim,
		&wallet.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("Failed to query wallet by user_id: %w", err)
	}

	return &wallet, nil
}

func (repository *postgresWalletRepository) GetWalletByID(context context.Context, walletID int64) (*domain.UserWallet, error) {
	query := `
		SELECT id, user_id, coin_balance, last_daily_claim, updated_at
		FROM wallet_service.wallets
		WHERE id = $1;
	`

	var wallet domain.UserWallet
	err := repository.pool.QueryRow(context, query, walletID).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.LastDailyClaim,
		&wallet.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("Failed to query wallet by ID: %w", err)
	}

	return &wallet, nil
}

//TODO
/*
	We take
*/
func (repository *postgresWalletRepository) ClaimDailyReward(context context.Context, userID int64, rewardAmount int64) (*domain.UserWallet, error) {
	transaction, err := repository.pool.Begin(context)
	if err != nil {
		return nil, fmt.Errorf("Failed to begin transaction: %w", err)
	}
	defer transaction.Rollback(context) //Ensures that after failing, any changes are automatically will be canceled

	lockQuery := `
		SELECT id, user_id, coin_balance, last_daily_claim, updated_at
		FROM wallet_service.wallets
		WHERE user_id = $1
		FOR UPDATE;
	`
	var wallet domain.UserWallet
	err = transaction.QueryRow(context, lockQuery, userID).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.LastDailyClaim,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("Failed to lock wallet row: %w", err)
	}

	if wallet.LastDailyClaim != nil {
		if time.Since(*wallet.LastDailyClaim) < 24*time.Hour {
			return nil, ErrDailyClaimAlreadyClaimed
		}
	}

	updatedQuery := `
		UPDATE wallet_service.wallets
		SET coin_balance = coin_balance + $1,
			last_daily_claim = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2
		RETURNING id, user_id, coin_balance, last_daily_claim, updated_at;
	`
	var updatedWallet domain.UserWallet
	err = transaction.QueryRow(context, updatedQuery, rewardAmount, userID).Scan(
		&updatedWallet.ID,
		&updatedWallet.UserID,
		&updatedWallet.Balance,
		&updatedWallet.LastDailyClaim,
		&updatedWallet.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to UPDATE wallet balance: %w", err)
	}

	if err := transaction.Commit(context); err != nil {
		return nil, fmt.Errorf("Failed to commit claim transaction: %w", err)
	}

	return &updatedWallet, nil
}
