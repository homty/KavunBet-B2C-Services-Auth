package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	userv1 "betwin/pkg/proto/user/v1"
	usergrpc "betwin/user-service/internal/API/grpc"
	"betwin/user-service/internal/domain"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type userServiceStub struct {
	provisionProfile func(context.Context, int64, string, string) (*domain.UserProfile, error)
	getPublicProfile func(context.Context, int64) (*domain.PublicProfileResponse, error)
}

var _ service.UserService = (*userServiceStub)(nil)

func (stub *userServiceStub) ProvisionProfile(ctx context.Context, userID int64, username, contactInfo string) (*domain.UserProfile, error) {
	if stub.provisionProfile == nil {
		panic("unexpected ProvisionProfile call")
	}
	return stub.provisionProfile(ctx, userID, username, contactInfo)
}

func (stub *userServiceStub) GetMyProfile(context.Context, int64) (*domain.UserProfile, error) {
	panic("unexpected GetMyProfile call")
}

func (stub *userServiceStub) UpdateMyProfile(context.Context, int64, domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	panic("unexpected UpdateMyProfile call")
}

func (stub *userServiceStub) GetPublicProfile(ctx context.Context, userID int64) (*domain.PublicProfileResponse, error) {
	if stub.getPublicProfile == nil {
		panic("unexpected GetPublicProfile call")
	}
	return stub.getPublicProfile(ctx, userID)
}

func newUserServiceClient(t *testing.T, userService service.UserService) userv1.UserServiceClient {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	userv1.RegisterUserServiceServer(server, usergrpc.NewUserGRPCHandler(userService))

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connection, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		server.Stop()
		t.Fatalf("dial in-memory gRPC server: %v", err)
	}

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close gRPC client connection: %v", err)
		}
		server.Stop()
		if err := <-serveErrors; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve gRPC API: %v", err)
		}
	})

	return userv1.NewUserServiceClient(connection)
}

func TestProvisionProfile(t *testing.T) {
	stub := &userServiceStub{
		provisionProfile: func(_ context.Context, userID int64, username, contactInfo string) (*domain.UserProfile, error) {
			if userID != 42 {
				t.Errorf("user ID = %d, want 42", userID)
			}
			if username != "alice" {
				t.Errorf("username = %q, want %q", username, "alice")
			}
			if contactInfo != "alice@example.com" {
				t.Errorf("contact info = %q, want %q", contactInfo, "alice@example.com")
			}
			return &domain.UserProfile{UserID: userID, Username: "stored-alice"}, nil
		},
	}
	client := newUserServiceClient(t, stub)

	response, err := client.ProvisionProfile(context.Background(), &userv1.ProvisionProfileRequest{
		UserId:      42,
		Username:    "alice",
		ContactInfo: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("ProvisionProfile returned an error: %v", err)
	}
	if response.GetUserId() != 42 {
		t.Errorf("response user ID = %d, want 42", response.GetUserId())
	}
	if response.GetUsername() != "stored-alice" {
		t.Errorf("response username = %q, want %q", response.GetUsername(), "stored-alice")
	}
}

func TestProvisionProfileRejectsInvalidRequests(t *testing.T) {
	handler := usergrpc.NewUserGRPCHandler(&userServiceStub{})
	tests := []struct {
		name    string
		request *userv1.ProvisionProfileRequest
		message string
	}{
		{name: "nil request", request: nil, message: "request is required"},
		{name: "invalid user ID", request: &userv1.ProvisionProfileRequest{Username: "alice"}, message: "user_id must be greater than zero"},
		{name: "blank username", request: &userv1.ProvisionProfileRequest{UserId: 42, Username: " \t"}, message: "username is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := handler.ProvisionProfile(context.Background(), test.request)
			assertGRPCError(t, err, codes.InvalidArgument, test.message)
		})
	}
}

func TestProvisionProfileHidesInternalServiceError(t *testing.T) {
	stub := &userServiceStub{
		provisionProfile: func(context.Context, int64, string, string) (*domain.UserProfile, error) {
			return nil, errors.New("insert profile: database password exposed")
		},
	}
	client := newUserServiceClient(t, stub)

	_, err := client.ProvisionProfile(context.Background(), &userv1.ProvisionProfileRequest{UserId: 42, Username: "alice"})
	assertGRPCError(t, err, codes.Internal, "internal server error")
}

func TestGetPublicProfile(t *testing.T) {
	displayName := "Alice Wonder"
	avatarURL := "https://example.com/alice.png"
	bio := "Prediction market enthusiast"
	stub := &userServiceStub{
		getPublicProfile: func(_ context.Context, userID int64) (*domain.PublicProfileResponse, error) {
			if userID != 42 {
				t.Errorf("target user ID = %d, want 42", userID)
			}
			return &domain.PublicProfileResponse{
				UserID:      userID,
				Username:    "alice",
				DisplayName: &displayName,
				AvatarURL:   &avatarURL,
				Bio:         &bio,
			}, nil
		},
	}
	client := newUserServiceClient(t, stub)

	response, err := client.GetPublicProfile(context.Background(), &userv1.GetPublicProfileRequest{TargetUserId: 42})
	if err != nil {
		t.Fatalf("GetPublicProfile returned an error: %v", err)
	}

	if response.GetUserId() != 42 {
		t.Errorf("response user ID = %d, want 42", response.GetUserId())
	}
	if response.GetUsername() != "alice" {
		t.Errorf("response username = %q, want %q", response.GetUsername(), "alice")
	}
	if response.GetDisplayName() != displayName {
		t.Errorf("response display name = %q, want %q", response.GetDisplayName(), displayName)
	}
	if response.GetAvatarUrl() != avatarURL {
		t.Errorf("response avatar URL = %q, want %q", response.GetAvatarUrl(), avatarURL)
	}
	if response.GetBio() != bio {
		t.Errorf("response bio = %q, want %q", response.GetBio(), bio)
	}
}

func TestGetPublicProfileRejectsInvalidRequests(t *testing.T) {
	handler := usergrpc.NewUserGRPCHandler(&userServiceStub{})
	tests := []struct {
		name    string
		request *userv1.GetPublicProfileRequest
		message string
	}{
		{name: "nil request", request: nil, message: "request is required"},
		{name: "invalid user ID", request: &userv1.GetPublicProfileRequest{}, message: "target_user_id must be greater than zero"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := handler.GetPublicProfile(context.Background(), test.request)
			assertGRPCError(t, err, codes.InvalidArgument, test.message)
		})
	}
}

func TestGetPublicProfileMapsNotFoundError(t *testing.T) {
	stub := &userServiceStub{
		getPublicProfile: func(context.Context, int64) (*domain.PublicProfileResponse, error) {
			return nil, repository.ErrProfileNotFound
		},
	}
	client := newUserServiceClient(t, stub)

	_, err := client.GetPublicProfile(context.Background(), &userv1.GetPublicProfileRequest{TargetUserId: 404})
	assertGRPCError(t, err, codes.NotFound, "profile not found")
}

func TestGetPublicProfileHidesInternalServiceError(t *testing.T) {
	stub := &userServiceStub{
		getPublicProfile: func(context.Context, int64) (*domain.PublicProfileResponse, error) {
			return nil, errors.New("query profile: database password exposed")
		},
	}
	client := newUserServiceClient(t, stub)

	_, err := client.GetPublicProfile(context.Background(), &userv1.GetPublicProfileRequest{TargetUserId: 42})
	assertGRPCError(t, err, codes.Internal, "internal server error")
}

func assertGRPCError(t *testing.T, err error, code codes.Code, message string) {
	t.Helper()
	if status.Code(err) != code {
		t.Fatalf("error code = %s, want %s; error = %v", status.Code(err), code, err)
	}
	if status.Convert(err).Message() != message {
		t.Errorf("error message = %q, want %q", status.Convert(err).Message(), message)
	}
}
