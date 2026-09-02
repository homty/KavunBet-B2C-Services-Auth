package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/domain"
	"betwin/auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials   = errors.New("Invalid username or password")
	ErrPasswordTooShort     = errors.New("Password must be at least 6 characters long")
	ErrUsernameRequired     = errors.New("Username is required")
	ErrUsernameTooShort     = errors.New("Username must be at least 3 characters long")
	ErrUsernameTooLong      = errors.New("Username cannot exceed 50 characters")
	ErrProfileProvisioning  = errors.New("Profile provisioning failed")
	ErrWalletProvisioning   = errors.New("Wallet provisioning failed")
	ErrRegistrationRollback = errors.New("Registration rollback failed")
)

const (
	minUsernameLength = 3
	maxUsernameLength = 50
)

type JWTClaims struct {
	jwt.RegisteredClaims
}

type AuthService interface {
	Register(context context.Context, request domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(context context.Context, request domain.LoginRequest) (*domain.AuthResponse, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
}

type ProfileProvisioner interface {
	ProvisionProfile(context.Context, int64, string, string) error
	DeleteProfile(context.Context, int64) error
}

type WalletProvisioner interface {
	ProvisionWallet(context.Context, int64) error
	DeleteWallet(context.Context, int64) error
}

type authService struct {
	userRepo           repository.UserRepository
	profileProvisioner ProfileProvisioner
	walletProvisioner  WalletProvisioner
	config             *config.Config
}

func NewAuthService(userRepo repository.UserRepository, profileProvisioner ProfileProvisioner, walletProvisioner WalletProvisioner, config *config.Config) AuthService {
	return &authService{
		userRepo:           userRepo,
		profileProvisioner: profileProvisioner,
		walletProvisioner:  walletProvisioner,
		config:             config,
	}
}

func (self *authService) Register(ctx context.Context, request domain.RegisterRequest) (*domain.AuthResponse, error) {
	request.Username = normalizeUsername(request.Username)
	if request.Username == "" {
		return nil, ErrUsernameRequired
	}
	usernameLength := utf8.RuneCountInString(request.Username)
	if usernameLength < minUsernameLength {
		return nil, ErrUsernameTooShort
	}
	if usernameLength > maxUsernameLength {
		return nil, ErrUsernameTooLong
	}
	if len(request.Password) < 6 {
		return nil, ErrPasswordTooShort
	}

	// Hashing password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Failed to hash password: %w", err)
	}

	// Save user into DB
	user, err := self.userRepo.CreatePendingUser(ctx, request.Username, string(hashedPassword))
	if err != nil {
		return nil, err
	}

	if self.profileProvisioner == nil {
		err = errors.New("profile provisioner is not configured")
	} else {
		err = self.profileProvisioner.ProvisionProfile(ctx, user.ID, user.Username, request.ContactInfo)
	}
	if err != nil {
		provisioningError := fmt.Errorf("%w: %w", ErrProfileProvisioning, err)
		if rollbackError := self.rollbackRegistration(ctx, user.ID, false); rollbackError != nil {
			return nil, errors.Join(provisioningError, rollbackError)
		}
		return nil, provisioningError
	}

	if self.walletProvisioner == nil {
		err = errors.New("wallet provisioner is not configured")
	} else {
		err = self.walletProvisioner.ProvisionWallet(ctx, user.ID)
	}
	if err != nil {
		provisioningError := fmt.Errorf("%w: %w", ErrWalletProvisioning, err)
		if rollbackError := self.rollbackRegistration(ctx, user.ID, true); rollbackError != nil {
			return nil, errors.Join(provisioningError, rollbackError)
		}
		return nil, provisioningError
	}

	if err := self.userRepo.UpdateUserStatus(ctx, user.ID, domain.AccountStatusActive); err != nil {
		activationError := fmt.Errorf("Failed to activate user: %w", err)
		if rollbackError := self.rollbackRegistration(ctx, user.ID, true); rollbackError != nil {
			return nil, errors.Join(activationError, rollbackError)
		}
		return nil, activationError
	}
	user.Status = domain.AccountStatusActive

	token, expiresIn, err := self.generateJWT(user.ID)
	if err != nil {
		tokenError := fmt.Errorf("Failed to generate token: %w", err)
		if rollbackError := self.rollbackRegistration(ctx, user.ID, true); rollbackError != nil {
			return nil, errors.Join(tokenError, rollbackError)
		}
		return nil, tokenError
	}

	return &domain.AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User: domain.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			CreatedAt: user.CreatedAt.UTC(),
		},
	}, nil
}

func (self *authService) rollbackRegistration(ctx context.Context, userID int64, cleanupWallet bool) error {
	cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	var rollbackErrors []error
	if err := self.userRepo.UpdateUserStatus(cleanupContext, userID, domain.AccountStatusFailed); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("mark user failed: %w", err))
	}

	downstreamCleanupSucceeded := true
	if cleanupWallet {
		if self.walletProvisioner == nil {
			downstreamCleanupSucceeded = false
			rollbackErrors = append(rollbackErrors, errors.New("wallet provisioner is not configured"))
		} else if err := self.walletProvisioner.DeleteWallet(cleanupContext, userID); err != nil {
			downstreamCleanupSucceeded = false
			rollbackErrors = append(rollbackErrors, fmt.Errorf("delete wallet: %w", err))
		}
	}

	if self.profileProvisioner == nil {
		downstreamCleanupSucceeded = false
		rollbackErrors = append(rollbackErrors, errors.New("profile provisioner is not configured"))
	} else if err := self.profileProvisioner.DeleteProfile(cleanupContext, userID); err != nil {
		downstreamCleanupSucceeded = false
		rollbackErrors = append(rollbackErrors, fmt.Errorf("delete profile: %w", err))
	}

	if downstreamCleanupSucceeded {
		if err := self.userRepo.DeleteUser(cleanupContext, userID); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("delete auth user: %w", err))
		}
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("%w: %w", ErrRegistrationRollback, errors.Join(rollbackErrors...))
	}
	return nil
}

func (self *authService) Login(context context.Context, request domain.LoginRequest) (*domain.AuthResponse, error) {
	request.Username = normalizeUsername(request.Username)
	if request.Username == "" || request.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := self.userRepo.GetUserByUsername(context, request.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("Failed to fetch user: %w", err)
	}
	if user.Status != domain.AccountStatusActive {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresIn, err := self.generateJWT(user.ID)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate token: %w", err)
	}

	return &domain.AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User: domain.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			CreatedAt: user.CreatedAt.UTC(),
		},
	}, nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func (self *authService) generateJWT(userID int64) (string, int64, error) {
	now := time.Now().UTC()
	expirationToken := now.Add(self.config.AccessJWTTTL)
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    "betwin-auth-service",
			Audience:  jwt.ClaimStrings{"betwin-api"},
			ExpiresAt: jwt.NewNumericDate(expirationToken),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(self.config.JWTSecret))
	if err != nil {
		return "", 0, fmt.Errorf("Failed to transform token into STRING: %w", err)
	}

	return tokenString, int64(self.config.AccessJWTTTL.Seconds()), nil
}

func (self *authService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(self.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("Invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("Invalid or expired token")
	}

	return claims, nil
}
