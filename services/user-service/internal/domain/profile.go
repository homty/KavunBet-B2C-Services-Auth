package domain

import "time"

// UserProfile represents full profile metadata stored in `user_profiles`
type UserProfile struct {
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	ContactInfo *string   `json:"contact_info,omitempty"`
	DisplayName *string   `json:"display_name,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateProfileRequest payload for PUT /api/v1/users/me
type UpdateProfileRequest struct {
	ContactInfo *string `json:"contact_info,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// PublicProfileResponse payload for GET /api/v1/users/{id}
type PublicProfileResponse struct {
	UserID      int64   `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}
