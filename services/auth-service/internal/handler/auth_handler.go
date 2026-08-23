package handler

import (
	"betwin/auth-service/internal/domain"
	"betwin/auth-service/internal/repository"
	"betwin/auth-service/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Setup APIs
func (handler *AuthHandler) RegisterRoutes(route chi.Router) {
	route.Use(middleware.Logger)
	route.Use(middleware.Recoverer)

	route.Get("/health", handler.HealthCheck)

	route.Route("/api/v1/auth", func(route chi.Router) {
		route.Post("/register", handler.Register)
		route.Post("/login", handler.Login)
		route.Get("/me", handler.GetProfile)
	})
}

// POST /auth/register
func (handler *AuthHandler) Register(write http.ResponseWriter, request *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		respondWithError(write, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload")
		return
	}

	response, err := handler.authService.Register(request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUsernameRequired) {
			respondWithError(write, http.StatusBadRequest, "USERNAME_REQUIRED", err.Error())
			return
		} else if errors.Is(err, service.ErrPasswordTooShort) {
			respondWithError(write, http.StatusBadRequest, "PASSWORD_TOO_SHORT", err.Error())
			return
		} else if errors.Is(err, repository.ErrDuplicateUsername) {
			respondWithError(write, http.StatusConflict, "USERNAME_ALREADY_EXISTS", "Username is already taken")
			return
		} else {
			respondWithError(write, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register user")
			return
		}
	}

	respondWithJSON(write, http.StatusCreated, response)
}

// POST /auth/login
func (handler *AuthHandler) Login(write http.ResponseWriter, request *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		respondWithError(write, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload")
		return
	}

	resp, err := handler.authService.Login(request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			respondWithError(write, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
			return
		}
		respondWithError(write, http.StatusInternalServerError, "INTERNAL_ERROR", "Login failed")
		return
	}

	respondWithJSON(write, http.StatusOK, resp)
}

// GET auth/me
func (handler *AuthHandler) GetProfile(write http.ResponseWriter, request *http.Request) {
	authHeader := request.Header.Get("Authorization")

	if authHeader == "" {
		respondWithError(write, http.StatusUnauthorized, "UNAUTHORIZED", "Missing Authorization header")
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		respondWithError(write, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid Authorization header format")
		return
	}

	tokenString := parts[1]
	claims, err := handler.authService.ValidateToken(tokenString)
	if err != nil {
		respondWithError(write, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
		return
	}

	respondWithJSON(write, http.StatusOK, map[string]interface{}{
		"user_id": claims.Subject,
	})
}

// Helper functions

// GET /health
func (handler *AuthHandler) HealthCheck(write http.ResponseWriter, requesst *http.Request) {
	respondWithJSON(write, http.StatusOK, map[string]string{
		"status": "UP",
	})
}

// Send JSON response
func respondWithJSON(write http.ResponseWriter, statusCode int, payload interface{}) {
	write.Header().Set("Content-Type", "application/json")
	write.WriteHeader(statusCode)
	_ = json.NewEncoder(write).Encode(payload)
}

// Structured error response
func respondWithError(w http.ResponseWriter, statusCode int, code string, message string) {
	respondWithJSON(w, statusCode, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
