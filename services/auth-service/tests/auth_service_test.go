package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/domain"
	"betwin/auth-service/internal/repository"
	"betwin/auth-service/internal/service"
)

// MockUserRepository simulates PostgreSQL in-memory for fast unit testing.
type MockUserRepository struct {
	users map[string]*domain.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *MockUserRepository) CreateUserWithProfileAndWallet(ctx context.Context, username, passwordHash, contactInfo string) (*domain.User, error) {
	if _, exists := m.users[username]; exists {
		return nil, repository.ErrDuplicateUsername
	}
	user := &domain.User{
		ID:           int64(len(m.users) + 1),
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[username] = user
	return user, nil
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	user, exists := m.users[username]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

// Setup helper for unit tests
func setupTestAuthService() service.AuthService {
	mockRepo := NewMockUserRepository()
	cfg := &config.Config{
		JWTSecret:    "unit_test_jwt_secret_key_12345",
		AccessJWTTTL: 15 * time.Minute,
	}
	return service.NewDBConnection(mockRepo, cfg)
}

func TestRegister_Success(t *testing.T) {
	authSvc := setupTestAuthService()
	ctx := context.Background()

	req := domain.RegisterRequest{
		Username:    "alice",
		Password:    "securepassword123",
		ContactInfo: "alice@example.com",
	}

	resp, err := authSvc.Register(ctx, req)
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}

	if resp.AccessToken == "" {
		t.Errorf("Expected non-empty AccessToken")
	}
	if resp.User.Username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", resp.User.Username)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	authSvc := setupTestAuthService()
	ctx := context.Background()

	req := domain.RegisterRequest{
		Username: "bob",
		Password: "123", // Too short
	}

	_, err := authSvc.Register(ctx, req)
	if !errors.Is(err, service.ErrPasswordTooShort) {
		t.Errorf("Expected ErrPasswordTooShort, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	authSvc := setupTestAuthService()
	ctx := context.Background()

	// 1. Register user
	regReq := domain.RegisterRequest{
		Username: "charlie",
		Password: "password123",
	}
	_, err := authSvc.Register(ctx, regReq)
	if err != nil {
		t.Fatalf("Failed to register user for login test: %v", err)
	}

	// 2. Login with correct password
	loginReq := domain.LoginRequest{
		Username: "charlie",
		Password: "password123",
	}
	loginResp, err := authSvc.Login(ctx, loginReq)
	if err != nil {
		t.Fatalf("Expected successful login, got error: %v", err)
	}

	if loginResp.AccessToken == "" {
		t.Errorf("Expected valid AccessToken upon login")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	authSvc := setupTestAuthService()
	ctx := context.Background()

	// Register user
	_, err := authSvc.Register(ctx, domain.RegisterRequest{
		Username: "dave",
		Password: "correctpassword",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Login with wrong password
	_, err = authSvc.Login(ctx, domain.LoginRequest{
		Username: "dave",
		Password: "wrongpassword",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials on wrong password, got %v", err)
	}
}

func TestValidateToken_SuccessAndInvalid(t *testing.T) {
	authSvc := setupTestAuthService()
	ctx := context.Background()

	// 1. Register & get token
	resp, err := authSvc.Register(ctx, domain.RegisterRequest{
		Username: "eve",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// 2. Validate token
	claims, err := authSvc.ValidateToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("Expected token to be valid, got error: %v", err)
	}
	if claims.Subject == "" {
		t.Errorf("Expected valid claims.Subject, got empty string")
	}

	// 3. Test invalid token string
	_, err = authSvc.ValidateToken("invalid.jwt.string")
	if err == nil {
		t.Errorf("Expected error when validating invalid token, got nil")
	}
}
