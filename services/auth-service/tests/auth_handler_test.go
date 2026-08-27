package tests

import (
	"betwin/auth-service/internal/domain"
	"betwin/auth-service/internal/handler"
	"betwin/auth-service/internal/service"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type authServiceSpy struct {
	registerCalled bool
	loginCalled    bool
}

var _ service.AuthService = (*authServiceSpy)(nil)

func (spy *authServiceSpy) Register(context.Context, domain.RegisterRequest) (*domain.AuthResponse, error) {
	spy.registerCalled = true
	return &domain.AuthResponse{}, nil
}

func (spy *authServiceSpy) Login(context.Context, domain.LoginRequest) (*domain.AuthResponse, error) {
	spy.loginCalled = true
	return &domain.AuthResponse{}, nil
}

func (spy *authServiceSpy) ValidateToken(string) (*service.JWTClaims, error) {
	return &service.JWTClaims{}, nil
}

func TestRegisterHandlerSuccess(t *testing.T) {
	spy := &authServiceSpy{}
	authHandler := handler.NewAuthHandler(spy)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"username":"alice","password":"secret"}`))
	recorder := httptest.NewRecorder()

	authHandler.Register(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if !spy.registerCalled {
		t.Fatal("expected handler to call Register")
	}
}

func TestRegisterHandlerRejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"username":"alice","password":"secret","admin":true}`},
		{name: "multiple objects", body: `{"username":"alice","password":"secret"}{"username":"bob","password":"secret"}`},
		{name: "too large", body: `{"username":"alice","password":"secret","contact_info":"` + strings.Repeat("a", (1<<20)+1) + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spy := &authServiceSpy{}
			authHandler := handler.NewAuthHandler(spy)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(test.body))
			recorder := httptest.NewRecorder()

			authHandler.Register(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
			}
			if spy.registerCalled {
				t.Fatal("service must not be called for an invalid request body")
			}
		})
	}
}

func TestLoginHandlerRejectsUnknownFields(t *testing.T) {
	spy := &authServiceSpy{}
	authHandler := handler.NewAuthHandler(spy)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"alice","password":"secret","remember_me":true}`))
	recorder := httptest.NewRecorder()

	authHandler.Login(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	if spy.loginCalled {
		t.Fatal("service must not be called for an invalid request body")
	}
}
