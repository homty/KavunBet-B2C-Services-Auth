package repository

import (
	"context"
	"errors"
	"fmt"

	"betwin/user-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("Profile Not Found")

type ProfileRepository interface {
	GetProfileByUserID(context context.Context, userID int64) (*domain.UserProfile, error)
	UpdateProfile(context context.Context, userID int64, request domain.UpdateProfileRequest) (*domain.UserProfile, error)
	GetPublicProfileByID(context context.Context, userID int64) (*domain.PublicProfileResponse, error)
}

type postgresProfileRepository struct {
	pool *pgxpool.Pool
}

func NewProfileRepository(pool *pgxpool.Pool) ProfileRepository {
	return &postgresProfileRepository{pool: pool}
}

func (repository *postgresProfileRepository) GetProfileByUserID(context context.Context, userID int64) (*domain.UserProfile, error) {
	query := `
		SELECT user_id, contact_info, display_name, bio, avatar_url, updated_at
		FROM user_profiles
		WHERE user_id = $1;
	`

	var profile domain.UserProfile

	err := repository.pool.QueryRow(context, query, userID).Scan(
		&profile.UserID,
		&profile.ContactInfo,
		&profile.DisplayName,
		&profile.Bio,
		&profile.AvatarURL,
		&profile.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("Failed to query user profile: %w", err)
	}

	return &profile, nil
}

func (repository *postgresProfileRepository) UpdateProfile(context context.Context, userID int64, request domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	existing, err := repository.GetProfileByUserID(context, userID)
	if err != nil {
		return nil, err
	}

	if request.ContactInfo != nil {
		existing.ContactInfo = request.ContactInfo
	}
	if request.DisplayName != nil {
		existing.DisplayName = request.DisplayName
	}
	if request.Bio != nil {
		existing.Bio = request.Bio
	}
	if request.AvatarURL != nil {
		existing.AvatarURL = request.AvatarURL
	}

	query := `
		UPDATE user_profiles
		SET contact_info = $1, display_name = $2, bio = $3, avatar_url = $4, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $5
		RETURNING user_id, contact_info, display_name, bio, avatar_url, updated_at;
	`

	var updated domain.UserProfile
	err = repository.pool.QueryRow(
		context,
		query,
		existing.ContactInfo,
		existing.DisplayName,
		existing.Bio,
		existing.AvatarURL,
		userID).Scan(
		&updated.UserID,
		&updated.ContactInfo,
		&updated.DisplayName,
		&updated.Bio,
		&updated.AvatarURL,
		&updated.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("Failed to UPDATE user profile; %w", err)
	}

	return &updated, nil
}

func (repository *postgresProfileRepository) GetPublicProfileByID(context context.Context, userID int64) (*domain.PublicProfileResponse, error) {
	query := `
		SELECT u.id, u.username, p.display_name, p.bio, p.avatar_url
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.id = $1
	`

	var publicProfile domain.PublicProfileResponse
	err := repository.pool.QueryRow(context, query, userID).Scan(
		&publicProfile.UserID,
		&publicProfile.Username,
		&publicProfile.DisplayName,
		&publicProfile.Bio,
		&publicProfile.AvatarURL,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("Failed to query PUBLIC profile: %w", err)
	}

	return &publicProfile, nil
}
