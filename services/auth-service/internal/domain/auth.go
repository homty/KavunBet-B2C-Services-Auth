package domain

import "time"

// RegisterRequest is what the user posts during signup.
type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	ContactInfo string `json:"contact_info,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse is the structure for API responses containing user information.
type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserResponse `json:"user"`
}
