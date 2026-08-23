package handler

import (
	"betwin/pkg/middleware"
	"betwin/pkg/response"
	"betwin/wallet-service/internal/repository"
	"betwin/wallet-service/internal/service"
	"errors"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"net/http"
	"strconv"
)

type WalletHandler struct {
	walletService service.WalletService
	jwtSecret     string
}

func NewWalletHandler(walletService service.WalletService, jwtSecret string) *WalletHandler {
	return &WalletHandler{
		walletService: walletService,
		jwtSecret:     jwtSecret,
	}
}

func (handler *WalletHandler) RegisterRoutes(router chi.Router) {
	router.Use(chiMiddleware.Logger)
	router.Use(chiMiddleware.Recoverer)

	router.Get("/health", handler.HealthCheck)

	router.Group(func(router chi.Router) {
		router.Use(middleware.AuthMiddleware(handler.jwtSecret))
		router.Get("/api/v1/wallet/balance", handler.GetBalance)
		router.Post("api/v1/wallet/daily-claim", handler.ClaimDaily)
		router.Get("/api/v1/wallet/{id}", handler.GetWalletByID)
	})
}

func (handler *WalletHandler) GetBalance(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(request.Context())
	if !ok {
		response.RespondWithError(writer, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	balanceRespond, err := handler.walletService.GetBalance(request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			response.RespondWithError(writer, http.StatusNotFound, "WALLET_NOT_FOUND", "Wallet not found")
			return
		}
		response.RespondWithError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch wallet balance")
		return
	}

	response.RespondWithJSON(writer, http.StatusOK, balanceRespond)
}

func (handler *WalletHandler) ClaimDaily(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(request.Context())
	if !ok {
		response.RespondWithError(writer, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	claimRepository, err := handler.walletService.ClaimDaily(request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrDailyClaimAlreadyClaimed) {
			response.RespondWithError(writer, http.StatusBadRequest, "DAILY_CLAIM_ALREADY_CLAIMED", err.Error())
			return
		}
		if errors.Is(err, repository.ErrWalletNotFound) {
			response.RespondWithError(writer, http.StatusNotFound, "WALLET_NOT_FOUND", "Wallet not found")
			return
		}
		response.RespondWithError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process daily claim")
		return
	}

	response.RespondWithJSON(writer, http.StatusOK, claimRepository)
}

func (handler *WalletHandler) GetWalletByID(writer http.ResponseWriter, request *http.Request) {
	idStr := chi.URLParam(request, "id")
	walletID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.RespondWithError(writer, http.StatusBadRequest, "INVALID_WALLET_ID", "Invalid wallet ID parameter")
		return
	}
	wallet, err := handler.walletService.GetWalletByID(request.Context(), walletID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			response.RespondWithError(writer, http.StatusNotFound, "WALLET_NOT_FOUND", "Wallet not found")
			return
		}
		response.RespondWithError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch wallet")
		return
	}
	response.RespondWithJSON(writer, http.StatusOK, wallet)
}

func (handler *WalletHandler) HealthCheck(writer http.ResponseWriter, request *http.Request) {
	response.RespondWithJSON(writer, http.StatusOK, map[string]string{
		"status": "UP",
	})
}
