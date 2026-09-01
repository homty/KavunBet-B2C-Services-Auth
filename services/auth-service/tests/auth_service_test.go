package tests

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/domain"
	"betwin/auth-service/internal/repository"
	"betwin/auth-service/internal/service"
)

// MockUserRepository simulates PostgreSQL in-memory for fast unit testing.
type MockUserRepository struct {
	users              map[string]*domain.User
	statusUpdateErrors map[domain.AccountStatus]error
	statusTransitions  []domain.AccountStatus
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:              make(map[string]*domain.User),
		statusUpdateErrors: make(map[domain.AccountStatus]error),
	}
}

func (m *MockUserRepository) CreatePendingUser(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	if _, exists := m.users[username]; exists {
		return nil, repository.ErrDuplicateUsername
	}
	user := &domain.User{
		ID:           int64(len(m.users) + 1),
		Username:     username,
		PasswordHash: passwordHash,
		Status:       domain.AccountStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[username] = user
	return user, nil
}

func (m *MockUserRepository) UpdateUserStatus(_ context.Context, userID int64, status domain.AccountStatus) error {
	if err := m.statusUpdateErrors[status]; err != nil {
		return err
	}
	for _, user := range m.users {
		if user.ID == userID {
			user.Status = status
			m.statusTransitions = append(m.statusTransitions, status)
			return nil
		}
	}
	return repository.ErrUserNotFound
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

type profileProvisionerStub struct {
	err              error
	called           bool
	receivedUserID   int64
	receivedUsername string
	receivedContact  string
	onProvision      func()
}

func (stub *profileProvisionerStub) ProvisionProfile(_ context.Context, userID int64, username, contactInfo string) error {
	stub.called = true
	stub.receivedUserID = userID
	stub.receivedUsername = username
	stub.receivedContact = contactInfo
	if stub.onProvision != nil {
		stub.onProvision()
	}
	return stub.err
}

// Setup helper for unit tests
func setupTestAuthService() service.AuthService {
	authSvc, _, _ := setupTestAuthServiceWithDependencies()
	return authSvc
}

func setupTestAuthServiceWithRepo() (service.AuthService, *MockUserRepository) {
	authSvc, mockRepo, _ := setupTestAuthServiceWithDependencies()
	return authSvc, mockRepo
}

func setupTestAuthServiceWithDependencies() (service.AuthService, *MockUserRepository, *profileProvisionerStub) {
	mockRepo := NewMockUserRepository()
	provisioner := &profileProvisionerStub{}
	cfg := &config.Config{
		JWTSecret:    "unit_test_jwt_secret_key_12345",
		AccessJWTTTL: 15 * time.Minute,
	}
	return service.NewAuthService(mockRepo, provisioner, cfg), mockRepo, provisioner
}

func TestRegister_Success(t *testing.T) {
	authSvc, mockRepo, provisioner := setupTestAuthServiceWithDependencies()
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
	if !provisioner.called {
		t.Fatal("Expected registration to provision the user profile")
	}
	if provisioner.receivedUserID != resp.User.ID || provisioner.receivedUsername != "alice" || provisioner.receivedContact != "alice@example.com" {
		t.Fatalf("Unexpected provisioning request: userID=%d username=%q contact=%q", provisioner.receivedUserID, provisioner.receivedUsername, provisioner.receivedContact)
	}
	if mockRepo.users["alice"].Status != domain.AccountStatusActive {
		t.Fatalf("Expected registered user to be ACTIVE, got %s", mockRepo.users["alice"].Status)
	}
	if len(mockRepo.statusTransitions) != 1 || mockRepo.statusTransitions[0] != domain.AccountStatusActive {
		t.Fatalf("Expected one transition to ACTIVE, got %v", mockRepo.statusTransitions)
	}
}

func TestRegister_ProvisionsWhilePending(t *testing.T) {
	authSvc, mockRepo, provisioner := setupTestAuthServiceWithDependencies()
	provisioner.onProvision = func() {
		if status := mockRepo.users["alice"].Status; status != domain.AccountStatusPending {
			t.Errorf("Status during provisioning = %s, want %s", status, domain.AccountStatusPending)
		}
	}

	_, err := authSvc.Register(context.Background(), domain.RegisterRequest{
		Username: "alice",
		Password: "securepassword123",
	})
	if err != nil {
		t.Fatalf("Expected successful registration, got %v", err)
	}
}

func TestRegister_ProvisioningFailureMarksUserFailedAndReturnsNoToken(t *testing.T) {
	authSvc, mockRepo, provisioner := setupTestAuthServiceWithDependencies()
	provisioner.err = errors.New("user service unavailable")

	response, err := authSvc.Register(context.Background(), domain.RegisterRequest{
		Username: "alice",
		Password: "securepassword123",
	})
	if !errors.Is(err, service.ErrProfileProvisioning) {
		t.Fatalf("Expected ErrProfileProvisioning, got %v", err)
	}
	if response != nil {
		t.Fatalf("Expected no auth response on provisioning failure, got %+v", response)
	}
	if status := mockRepo.users["alice"].Status; status != domain.AccountStatusFailed {
		t.Fatalf("Status after provisioning failure = %s, want %s", status, domain.AccountStatusFailed)
	}
	if len(mockRepo.statusTransitions) != 1 || mockRepo.statusTransitions[0] != domain.AccountStatusFailed {
		t.Fatalf("Expected one transition to FAILED, got %v", mockRepo.statusTransitions)
	}
}

func TestRegister_ActivationFailureReturnsNoTokenAndMarksUserFailed(t *testing.T) {
	authSvc, mockRepo, _ := setupTestAuthServiceWithDependencies()
	mockRepo.statusUpdateErrors[domain.AccountStatusActive] = errors.New("database unavailable")

	response, err := authSvc.Register(context.Background(), domain.RegisterRequest{
		Username: "alice",
		Password: "securepassword123",
	})
	if err == nil {
		t.Fatal("Expected activation failure")
	}
	if response != nil {
		t.Fatalf("Expected no auth response before activation, got %+v", response)
	}
	if status := mockRepo.users["alice"].Status; status != domain.AccountStatusFailed {
		t.Fatalf("Status after activation failure = %s, want %s", status, domain.AccountStatusFailed)
	}
	if len(mockRepo.statusTransitions) != 1 || mockRepo.statusTransitions[0] != domain.AccountStatusFailed {
		t.Fatalf("Expected one transition to FAILED, got %v", mockRepo.statusTransitions)
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

func TestRegister_NormalizesUsername(t *testing.T) {
	authSvc := setupTestAuthService()

	resp, err := authSvc.Register(context.Background(), domain.RegisterRequest{
		Username: "  Alice  ",
		Password: "securepassword123",
	})
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}
	if resp.User.Username != "alice" {
		t.Fatalf("Expected normalized username %q, got %q", "alice", resp.User.Username)
	}
}

func TestRegister_UsernameLength(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{name: "too short", username: "ab", wantErr: service.ErrUsernameTooShort},
		{name: "too long", username: strings.Repeat("a", 51), wantErr: service.ErrUsernameTooLong},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authSvc := setupTestAuthService()
			_, err := authSvc.Register(context.Background(), domain.RegisterRequest{
				Username: test.username,
				Password: "securepassword123",
			})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestLogin_NormalizesUsername(t *testing.T) {
	authSvc := setupTestAuthService()
	ctx := context.Background()

	_, err := authSvc.Register(ctx, domain.RegisterRequest{
		Username: "alice",
		Password: "securepassword123",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	_, err = authSvc.Login(ctx, domain.LoginRequest{
		Username: "  ALICE  ",
		Password: "securepassword123",
	})
	if err != nil {
		t.Fatalf("Expected normalized login to succeed, got %v", err)
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

func TestLogin_PendingAccountRejected(t *testing.T) {
	authSvc, mockRepo := setupTestAuthServiceWithRepo()
	ctx := context.Background()

	_, err := authSvc.Register(ctx, domain.RegisterRequest{
		Username: "pending-user",
		Password: "correctpassword",
	})
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	mockRepo.users["pending-user"].Status = domain.AccountStatusPending

	_, err = authSvc.Login(ctx, domain.LoginRequest{
		Username: "pending-user",
		Password: "correctpassword",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("Expected pending account login to be rejected, got %v", err)
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
