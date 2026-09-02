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
	CreatePendingUser(context context.Context, username, passwordHash string) (*domain.User, error)
	UpdateUserStatus(context context.Context, userID int64, status domain.AccountStatus) error
	DeleteUser(context context.Context, userID int64) error
	GetUserByUsername(context context.Context, username string) (*domain.User, error)
	GetUserByID(context context.Context, userID int64) (*domain.User, error)
}

func (r *postgresUserRepository) DeleteUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM auth_service.users WHERE id = $1;`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) CreatePendingUser(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	createUserQuery := `
		INSERT INTO auth_service.users (username, password_hash, status)
		VALUES ($1, $2, 'PENDING')
		RETURNING id, username, password_hash, status, created_at, updated_at;
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, createUserQuery, username, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
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

	return &user, nil
}

func (r *postgresUserRepository) UpdateUserStatus(ctx context.Context, userID int64, accountStatus domain.AccountStatus) error {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE auth_service.users
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2;
	`, accountStatus, userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (read *postgresUserRepository) GetUserByUsername(context context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash, status
		FROM auth_service.users
		WHERE username = $1;
	`

	var user domain.User
	err := read.pool.QueryRow(context, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
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
		SELECT id, username, password_hash, status
		FROM auth_service.users
		WHERE id = $1;
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("error querying user by id: %w", err)
	}
	return &user, nil
}
