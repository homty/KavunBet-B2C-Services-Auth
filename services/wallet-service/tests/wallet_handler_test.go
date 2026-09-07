package tests

import (
	"betwin/wallet-service/internal/API/rest/handler"
	"betwin/wallet-service/internal/domain"
	"betwin/wallet-service/internal/middleware"
	"betwin/wallet-service/internal/service"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

var testJWTPrivateKey = ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
var testJWTPublicKey = testJWTPrivateKey.Public().(ed25519.PublicKey)

type walletServiceSpy struct {
	getBalanceCalled   bool
	claimDailyCalled   bool
	receivedUserID     int64
	getBalanceResponse *domain.WalletBalanceResponse
	getBalanceError    error
	claimDailyResponse *domain.DailyClaimResponse
	claimDailyError    error
}

var _ service.WalletService = (*walletServiceSpy)(nil)

func (spy *walletServiceSpy) ProvisionWallet(_ context.Context, userID int64) (*domain.UserWallet, error) {
	spy.receivedUserID = userID
	return &domain.UserWallet{UserID: userID, Balance: 1000}, nil
}

func (spy *walletServiceSpy) DeleteWallet(_ context.Context, userID int64) error {
	spy.receivedUserID = userID
	return nil
}

func (spy *walletServiceSpy) GetBalance(_ context.Context, userID int64) (*domain.WalletBalanceResponse, error) {
	spy.getBalanceCalled = true
	spy.receivedUserID = userID
	if spy.getBalanceResponse != nil || spy.getBalanceError != nil {
		return spy.getBalanceResponse, spy.getBalanceError
	}
	return &domain.WalletBalanceResponse{UserID: userID, Balance: 100}, nil
}

func (spy *walletServiceSpy) ClaimDaily(_ context.Context, userID int64) (*domain.DailyClaimResponse, error) {
	spy.claimDailyCalled = true
	spy.receivedUserID = userID
	if spy.claimDailyResponse != nil || spy.claimDailyError != nil {
		return spy.claimDailyResponse, spy.claimDailyError
	}
	return &domain.DailyClaimResponse{}, nil
}

func TestWalletRoutesDoNotExposeWalletByID(t *testing.T) {
	router := chi.NewRouter()
	walletHandler := handler.NewWalletHandler(&walletServiceSpy{}, testJWTPublicKey)
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
	walletHandler := handler.NewWalletHandler(spy, testJWTPublicKey)
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

func TestGetBalanceRequiresAuthentication(t *testing.T) {
	spy := &walletServiceSpy{}
	router := chi.NewRouter()
	handler.NewWalletHandler(spy, testJWTPublicKey).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	if spy.getBalanceCalled {
		t.Fatal("wallet service must not be called when authentication fails")
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected error code UNAUTHORIZED, got %q", body.Error.Code)
	}
}

func TestGetBalanceWithValidToken(t *testing.T) {
	const (
		userID = int64(41)
	)

	expected := &domain.WalletBalanceResponse{
		WalletID:      7,
		UserID:        userID,
		Balance:       1000,
		CanClaimDaily: true,
	}
	spy := &walletServiceSpy{getBalanceResponse: expected}
	router := chi.NewRouter()
	handler.NewWalletHandler(spy, testJWTPublicKey).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	request.Header.Set("Authorization", "Bearer "+signWalletTestToken(t, userID))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !spy.getBalanceCalled {
		t.Fatal("expected handler to call WalletService.GetBalance")
	}
	if spy.receivedUserID != userID {
		t.Fatalf("expected authenticated user ID %d, got %d", userID, spy.receivedUserID)
	}

	var response domain.WalletBalanceResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode balance response: %v", err)
	}
	if response.WalletID != expected.WalletID || response.UserID != expected.UserID || response.Balance != expected.Balance || response.CanClaimDaily != expected.CanClaimDaily {
		t.Fatalf("unexpected balance response: %#v", response)
	}
}

func TestClaimDailyRequiresAuthentication(t *testing.T) {
	spy := &walletServiceSpy{}
	router := chi.NewRouter()
	handler.NewWalletHandler(spy, testJWTPublicKey).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/daily-claim", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	if spy.claimDailyCalled {
		t.Fatal("wallet service must not be called when authentication fails")
	}
}

func TestClaimDailyWithValidToken(t *testing.T) {
	const (
		userID = int64(41)
	)

	expected := &domain.DailyClaimResponse{
		Message:      "Successfully claimed +100 daily bonus coins!",
		WalletID:     7,
		CoinsClaimed: 100,
		NewBalance:   1100,
	}
	spy := &walletServiceSpy{claimDailyResponse: expected}
	router := chi.NewRouter()
	handler.NewWalletHandler(spy, testJWTPublicKey).RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/daily-claim", nil)
	request.Header.Set("Authorization", "Bearer "+signWalletTestToken(t, userID))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if !spy.claimDailyCalled {
		t.Fatal("expected handler to call WalletService.ClaimDaily")
	}
	if spy.receivedUserID != userID {
		t.Fatalf("expected authenticated user ID %d, got %d", userID, spy.receivedUserID)
	}

	var response domain.DailyClaimResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode daily-claim response: %v", err)
	}
	if response.WalletID != expected.WalletID || response.CoinsClaimed != expected.CoinsClaimed || response.NewBalance != expected.NewBalance || response.Message != expected.Message {
		t.Fatalf("unexpected daily-claim response: %#v", response)
	}
}

func signWalletTestToken(t *testing.T, userID int64) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub": strconv.FormatInt(userID, 10),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "betwin-auth-service",
		"aud": "betwin-api",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signedToken, err := token.SignedString(testJWTPrivateKey)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signedToken
}
