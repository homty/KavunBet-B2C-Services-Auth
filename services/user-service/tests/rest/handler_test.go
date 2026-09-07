package rest_test

import (
	"betwin/user-service/internal/API/rest/handler"
	"betwin/user-service/internal/domain"
	"betwin/user-service/internal/middleware"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testJWTPublicKey = ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public().(ed25519.PublicKey)

type userServiceSpy struct {
	updateCalled bool

	profileData       *domain.UserProfile
	getMyProfileError error

	receivedUserID  int64
	receivedRequest domain.UpdateProfileRequest
}

var _ service.UserService = (*userServiceSpy)(nil)

func (spy *userServiceSpy) ProvisionProfile(context.Context, int64, string, string) (*domain.UserProfile, error) {
	panic("unexpected ProvisionProfile call")
}

func (spy *userServiceSpy) DeleteProfile(context.Context, int64) error {
	panic("unexpected DeleteProfile call")
}

func (spy *userServiceSpy) UpdateMyProfile(context context.Context, userID int64, request domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	spy.updateCalled = true
	spy.receivedUserID = userID
	spy.receivedRequest = request

	return &domain.UserProfile{}, nil
}

func (spy *userServiceSpy) GetMyProfile(
	ctx context.Context,
	userID int64,
) (*domain.UserProfile, error) {
	return spy.profileData, spy.getMyProfileError
}

func (spy *userServiceSpy) GetPublicProfile(
	ctx context.Context,
	userID int64,
) (*domain.PublicProfileResponse, error) {
	panic("unexpected GetPublicProfile call")
}

func TestGetMyProfileNotFound(t *testing.T) {
	spy := &userServiceSpy{
		getMyProfileError: repository.ErrProfileNotFound,
	}
	userHandler := handler.NewUserHandler(spy, testJWTPublicKey)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)
	ctx := context.WithValue(
		request.Context(),
		middleware.UserIDKey,
		int64(52),
	)
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	userHandler.GetMyProfile(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestGetMyProfileSuccess(t *testing.T) {
	displayName := "TESTER"

	profile := domain.UserProfile{
		UserID:      52,
		DisplayName: &displayName,
	}
	spy := &userServiceSpy{
		profileData: &profile,
	}
	userHandler := handler.NewUserHandler(spy, testJWTPublicKey)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)
	ctx := context.WithValue(
		request.Context(),
		middleware.UserIDKey,
		int64(52),
	)
	request = request.WithContext(ctx)

	recorder := httptest.NewRecorder()
	userHandler.GetMyProfile(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Handler wasnt successfull. The ERROR code: %d", recorder.Code)
	}

	var decodedProfile domain.UserProfile

	err := json.NewDecoder(recorder.Body).Decode(&decodedProfile)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if decodedProfile.UserID != 52 {
		t.Fatalf("MUST EQUAL TO 52, NOW VALUE IS: %d", decodedProfile.UserID)
	}
	if decodedProfile.DisplayName == nil {
		t.Fatalf("DisplayName is NIL")
	}
	if *decodedProfile.DisplayName != "TESTER" {
		t.Fatalf("The displayName is dismatched with TESTER: %s", *decodedProfile.DisplayName)
	}
}

func TestGetMyProfileInternalError(t *testing.T) {
	spy := &userServiceSpy{
		getMyProfileError: errors.New("database unavailable"),
	}
	userHandler := handler.NewUserHandler(spy, testJWTPublicKey)

	request := httptest.NewRequest( //Request (method, path, body)
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	ctx := context.WithValue(
		request.Context(),
		middleware.UserIDKey,
		int64(52),
	)
	request = request.WithContext(ctx)

	recorder := httptest.NewRecorder()
	userHandler.GetMyProfile(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("500 Internal Server Error")
	}

	if !json.Valid(recorder.Body.Bytes()) {
		t.Fatalf("JSON form return false, JSON FORMAT: %q", recorder.Body.String())
	}
}
func TestUpdateMyProfile(t *testing.T) {
	spy := &userServiceSpy{}
	userHandler := handler.NewUserHandler(spy, testJWTPublicKey)

	body := strings.NewReader(`{"display_name":"TESTER"}`)

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/users/me",
		body,
	)

	ctx := context.WithValue(
		request.Context(),
		middleware.UserIDKey,
		int64(42),
	)
	request = request.WithContext(ctx)

	recorder := httptest.NewRecorder()
	userHandler.UpdateMyProfile(recorder, request)

	if !spy.updateCalled {
		t.Fatalf(
			"expected handler to call UpdateMyProfile; status=%d body=%s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if spy.receivedRequest.DisplayName == nil {
		t.Fatal("expected display_name to reach the service, got nil")
	}

	if *spy.receivedRequest.DisplayName != "TESTER" {
		t.Errorf(
			"expected display_name %q, got %q",
			"TESTER",
			*spy.receivedRequest.DisplayName,
		)
	}
}
