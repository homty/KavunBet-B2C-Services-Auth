package repository

import (
	"betwin/auth-service/internal/domain"

	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDuplicateUsername = errors.New("Username already exists")
	ErrUserNotFound      = errors.New("User not found")
)

type UserRepository interface {
	CreateUserWithProfileAndWallet(context context.Context, username, passwordHash, contactInfo string) (*domain.User, error)
	GetUserByUsername(context context.Context, username string) (*domain.User, error)
	GetUserByID(context context.Context, userID int64) (*domain.User, error)
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) CreateUserWithProfileAndWallet(ctx context.Context, username, passwordHash, contactInfo string) (*domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert into `users` table
	createUserQuery := `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, username, password_hash, created_at, updated_at;
	`
	var user domain.User
	err = tx.QueryRow(ctx, createUserQuery, username, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateUsername
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	var contactPtr *string
	if contactInfo != "" {
		contactPtr = &contactInfo
	}
	createProfileQuery := `
		INSERT INTO user_profiles (user_id, contact_info)
		VALUES ($1, $2);
	`
	_, err = tx.Exec(ctx, createProfileQuery, user.ID, contactPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user profile: %w", err)
	}

	createWalletQuery := `
		INSERT INTO user_wallets (user_id, coin_balance)
		VALUES ($1, 1000);
	`
	_, err = tx.Exec(ctx, createWalletQuery, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user wallet: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &user, nil
}

func (read *postgresUserRepository) GetUserByUsername(context context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash FROM users WHERE username = $1;
	`

	var user domain.User
	err := read.pool.QueryRow(context, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		} else {
			return nil, fmt.Errorf("Error querying user by username: %w", err)
		}
	}
	return &user, nil
}

func (r *postgresUserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash
		FROM users
		WHERE id = $1;
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("error querying user by id: %w", err)
	}
	return &user, nil
}
