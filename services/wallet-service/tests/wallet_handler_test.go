package tests

import (
	"betwin/pkg/middleware"
	"betwin/wallet-service/internal/domain"
	"betwin/wallet-service/internal/handler"
	"betwin/wallet-service/internal/service"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type walletServiceSpy struct {
	receivedUserID int64
}

var _ service.WalletService = (*walletServiceSpy)(nil)

func (spy *walletServiceSpy) ProvisionWallet(_ context.Context, userID int64) (*domain.UserWallet, error) {
	spy.receivedUserID = userID
	return &domain.UserWallet{UserID: userID, Balance: 1000}, nil
}

func (spy *walletServiceSpy) GetBalance(_ context.Context, userID int64) (*domain.WalletBalanceResponse, error) {
	spy.receivedUserID = userID
	return &domain.WalletBalanceResponse{UserID: userID, Balance: 100}, nil
}

func (spy *walletServiceSpy) ClaimDaily(_ context.Context, userID int64) (*domain.DailyClaimResponse, error) {
	spy.receivedUserID = userID
	return &domain.DailyClaimResponse{}, nil
}

func TestWalletRoutesDoNotExposeWalletByID(t *testing.T) {
	router := chi.NewRouter()
	walletHandler := handler.NewWalletHandler(&walletServiceSpy{}, "test-secret")
	walletHandler.RegisterRoutes(router)

	routes := make(map[string]bool)
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("failed to inspect routes: %v", err)
	}

	if !routes[http.MethodPost+" /api/v1/wallet/daily-claim"] {
		t.Fatal("daily-claim route is missing or malformed")
	}
	if routes[http.MethodGet+" /api/v1/wallet/{id}"] {
		t.Fatal("wallet-by-ID route must not be exposed")
	}
}

func TestGetBalanceUsesAuthenticatedUserID(t *testing.T) {
	spy := &walletServiceSpy{}
	walletHandler := handler.NewWalletHandler(spy, "test-secret")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	request = request.WithContext(context.WithValue(request.Context(), middleware.UserIDKey, int64(41)))
	recorder := httptest.NewRecorder()

	walletHandler.GetBalance(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if spy.receivedUserID != 41 {
		t.Fatalf("expected authenticated user ID %d, got %d", 41, spy.receivedUserID)
	}
}
