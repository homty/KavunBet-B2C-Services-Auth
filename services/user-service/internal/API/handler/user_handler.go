package handler

import (
	"betwin/user-service/internal/API/response"
	"betwin/user-service/internal/domain"
	"betwin/user-service/internal/middleware"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type UserHandler struct {
	userService service.UserService
	jwtSecret   string
}

func NewUserHandler(userService service.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

func (handler *UserHandler) RegisterRoutes(router chi.Router) {
	router.Use(chiMiddleware.Logger)
	router.Use(chiMiddleware.Recoverer)

	router.Get("/health", handler.HealthCheck)
	router.Get("/api/v1/users/{id}", handler.GetPublicProfile)

	//Isolated APIs from user view
	router.Group(func(router chi.Router) {
		router.Use(middleware.AuthMiddleware(handler.jwtSecret))
		router.Put("/api/v1/users/me", handler.UpdateMyProfile)
		router.Get("/api/v1/users/me", handler.GetMyProfile)
	})
}

func (handler *UserHandler) GetMyProfile(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(request.Context())
	if !ok {
		response.RespondWithError(writer, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	profile, err := handler.userService.GetMyProfile(request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			response.RespondWithError(writer, http.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
			return
		}
		response.RespondWithError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch profile")
	}

	response.RespondWithJSON(writer, http.StatusOK, profile)
}

func (handler *UserHandler) UpdateMyProfile(writer http.ResponseWriter, request *http.Request) {
	//Middleware checker
	userID, ok := middleware.GetUserIDFromContext(request.Context())
	if !ok {
		response.RespondWithError(writer, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var requestDomain domain.UpdateProfileRequest
	if err := json.NewDecoder(request.Body).Decode(&request); err != nil {
		response.RespondWithError(writer, http.StatusBadRequest, "INVALID_JSON", "Invalide JSON body")
		return
	}

	updatedProfile, err := handler.userService.UpdateMyProfile(request.Context(), userID, requestDomain)
	if err != nil {
		if errors.Is(err, service.ErrDisplayNameTooLong) ||
			errors.Is(err, service.ErrBioTooLong) ||
			errors.Is(err, service.ErrAvatarURLTooLong) {

			response.RespondWithError(writer, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return

		} else if errors.Is(err, repository.ErrProfileNotFound) {

			response.RespondWithError(writer, http.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
			return

		} else {

			response.RespondWithError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update profile")
			return

		}
	}

	response.RespondWithJSON(writer, http.StatusOK, updatedProfile)
}

func (handler *UserHandler) GetPublicProfile(writer http.ResponseWriter, request *http.Request) {
	idStr := chi.URLParam(request, "id")
	targetUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.RespondWithError(writer, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID parameter")
		return
	}

	publicProfile, err := handler.userService.GetPublicProfile(request.Context(), targetUserID)

	if err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			response.RespondWithError(writer, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
			return
		} else {
			response.RespondWithError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch public profile")
			return
		}
	}

	response.RespondWithJSON(writer, http.StatusOK, publicProfile)
}

func (handler *UserHandler) HealthCheck(writer http.ResponseWriter, request *http.Request) {
	response.RespondWithJSON(writer, http.StatusOK, map[string]string{
		"status": "UP",
	})
}
