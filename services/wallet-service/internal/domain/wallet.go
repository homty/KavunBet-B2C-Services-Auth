package domain

import "time"

type UserWallet struct {
	ID             int64      `json:"wallet_id"`
	UserID         int64      `json:"user_id"`
	Balance        int64      `json:"balance"`
	LastDailyClaim *time.Time `json:"last_daily_claim,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type WalletBalanceResponse struct {
	WalletID       int64      `json:"wallet_id"`
	UserID         int64      `json:"user_id"`
	Balance        int64      `json:"balance"`
	LastDailyClaim *time.Time `json:"last_daily_claim,omitempty"`
	CanClaimDaily  bool       `json:"can_claim_daily"`
}

type DailyClaimResponse struct {
	Message              string    `json:"message"`
	WalletID             int64     `json:"wallet_id"`
	CoinsClaimed         int64     `json:"coins_claimed"`
	NewBalance           int64     `json:"new_balance"`
	NextClaimAvailableAt time.Time `json:"next_claim_available_at"`
}
