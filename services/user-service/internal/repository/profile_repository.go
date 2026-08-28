package repository

import (
	"context"
	"errors"
	"fmt"

	"betwin/user-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound     = errors.New("Profile Not Found")
	ErrProfileIsDuplicated = errors.New("Duplicated Profile")
)

type ProfileRepository interface {
	ProvisionProfile(context context.Context, userID int64, username string, contactInfo string) (*domain.UserProfile, error)
	GetProfileByUserID(context context.Context, userID int64) (*domain.UserProfile, error)
	UpdateProfile(context context.Context, userID int64, request domain.UpdateProfileRequest) (*domain.UserProfile, error)
	GetPublicProfileByID(context context.Context, userID int64) (*domain.PublicProfileResponse, error)
}

func (repository *postgresProfileRepository) ProvisionProfile(context context.Context, userID int64, username string, contactInfo string) (*domain.UserProfile, error) {
	var contactPtr *string
	if contactInfo != "" {
		contactPtr = &contactInfo
	}

	query := `
		INSERT INTO user_profiles (user_id, username, contact_info)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO NOTHING;
	`
	if _, err := repository.pool.Exec(context, query, userID, username, contactPtr); err != nil {
		return nil, fmt.Errorf("failed to provision user profile: %w", err)
	}

	return repository.GetProfileByUserID(context, userID)
}

type postgresProfileRepository struct {
	pool *pgxpool.Pool
}

func NewProfileRepository(pool *pgxpool.Pool) ProfileRepository {
	return &postgresProfileRepository{pool: pool}
}

func (repository *postgresProfileRepository) InitializeProfile(ctx context.Context, userID int64, username, contactInfo string) (*domain.UserProfile, error) {
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to establish pool")
	}
	defer transaction.Rollback(ctx) //drop transaction if it has error

	query := `INSER INTO user_profiles (user_id, display_name, contact_info)
	VALUES ($1, $2, $3);`

	var profile domain.UserProfile
	err = transaction.QueryRow(ctx, query, userID, username, contactInfo).Scan(
		&profile.UserID,
		&profile.DisplayName,
		&profile.ContactInfo,
		&profile.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrProfileIsDuplicated
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return &profile, nil
}

func (repository *postgresProfileRepository) GetProfileByUserID(context context.Context, userID int64) (*domain.UserProfile, error) {
	query := `
		SELECT user_id, username, contact_info, display_name, bio, avatar_url, updated_at
		FROM user_profiles
		WHERE user_id = $1;
	`

	var profile domain.UserProfile

	err := repository.pool.QueryRow(context, query, userID).Scan(
		&profile.UserID,
		&profile.Username,
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
		RETURNING user_id, username, contact_info, display_name, bio, avatar_url, updated_at;
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
		&updated.Username,
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
		SELECT user_id, username, display_name, bio, avatar_url
		FROM user_profiles
		WHERE user_id = $1
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
