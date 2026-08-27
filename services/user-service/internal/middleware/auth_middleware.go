package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"betwin/user-service/internal/API/rest/response"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userID"

// GetUserIDFromContext helper function to extract user ID inside HTTP handlers
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}

// AuthMiddleware verifies the JWT access token and injects userID into request context.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing Authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid Authorization header format")
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				response.RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
				return
			}

			// Extract 'sub' claim (user ID string)
			sub, ok := claims["sub"].(string)
			if !ok {
				response.RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing sub claim")
				return
			}

			userID, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				response.RespondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user ID in token")
				return
			}

			// Inject userID into context and proceed to next handler
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
