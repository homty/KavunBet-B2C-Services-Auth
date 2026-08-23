package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"betwin/auth-service/internal/config"
	"betwin/auth-service/internal/domain"
	"betwin/auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("Invalid username or password")
	ErrPasswordTooShort   = errors.New("Password must be at least 6 characters long")
	ErrUsernameRequired   = errors.New("Username is required")
)

type JWTClaims struct {
	jwt.RegisteredClaims
}

type AuthService interface {
	Register(context context.Context, request domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(context context.Context, request domain.LoginRequest) (*domain.AuthResponse, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
}

type authService struct {
	userRepo repository.UserRepository
	config   *config.Config
}

func NewDBConnection(userRepo repository.UserRepository, config *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		config:   config,
	}
}

func (self *authService) Register(context context.Context, request domain.RegisterRequest) (*domain.AuthResponse, error) {
	if request.Username == "" {
		return nil, ErrUsernameRequired
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
	user, err := self.userRepo.CreateUserWithProfileAndWallet(context, request.Username, string(hashedPassword), request.ContactInfo)
	if err != nil {
		return nil, err
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

func (self *authService) Login(context context.Context, request domain.LoginRequest) (*domain.AuthResponse, error) {
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
