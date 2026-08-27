package domain

import "time"

type AccountStatus string

const (
	AccountStatusPending AccountStatus = "PENDING"
	AccountStatusActive  AccountStatus = "ACTIVE"
	AccountStatusFailed  AccountStatus = "FAILED"
)

type User struct {
	ID           int64         `json:"id"`
	Username     string        `json:"username"`
	PasswordHash string        `json:"-"` // Exclude from JSON output
	Status       AccountStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}
