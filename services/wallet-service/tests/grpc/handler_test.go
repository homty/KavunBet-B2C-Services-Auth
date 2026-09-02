package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	walletv1 "betwin/pkg/proto/wallet/v1"
	walletgrpc "betwin/wallet-service/internal/API/grpc/handler"
	"betwin/wallet-service/internal/domain"
	"betwin/wallet-service/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type walletServiceStub struct {
	provisionWallet func(context.Context, int64) (*domain.UserWallet, error)
	deleteWallet    func(context.Context, int64) error
	getBalance      func(context.Context, int64) (*domain.WalletBalanceResponse, error)
}

var _ service.WalletService = (*walletServiceStub)(nil)

func (stub *walletServiceStub) ProvisionWallet(ctx context.Context, userID int64) (*domain.UserWallet, error) {
	if stub.provisionWallet == nil {
		panic("unexpected ProvisionWallet call")
	}
	return stub.provisionWallet(ctx, userID)
}

func (stub *walletServiceStub) DeleteWallet(ctx context.Context, userID int64) error {
	if stub.deleteWallet == nil {
		panic("unexpected DeleteWallet call")
	}
	return stub.deleteWallet(ctx, userID)
}

func (stub *walletServiceStub) GetBalance(ctx context.Context, userID int64) (*domain.WalletBalanceResponse, error) {
	if stub.getBalance == nil {
		panic("unexpected GetBalance call")
	}
	return stub.getBalance(ctx, userID)
}

func (stub *walletServiceStub) ClaimDaily(context.Context, int64) (*domain.DailyClaimResponse, error) {
	panic("unexpected ClaimDaily call")
}

func newWalletServiceClient(t *testing.T, walletService service.WalletService) walletv1.WalletServiceClient {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	walletv1.RegisterWalletServiceServer(server, walletgrpc.NewWalletGRPCHandler(walletService))

	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.Serve(listener) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		server.Stop()
		t.Fatalf("dial in-memory gRPC server: %v", err)
	}

	t.Cleanup(func() {
		_ = connection.Close()
		server.Stop()
		if err := <-serveErrors; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve wallet gRPC API: %v", err)
		}
	})

	return walletv1.NewWalletServiceClient(connection)
}

func TestProvisionWalletReturnsPersistedWallet(t *testing.T) {
	stub := &walletServiceStub{
		provisionWallet: func(_ context.Context, userID int64) (*domain.UserWallet, error) {
			if userID != 42 {
				t.Errorf("user ID = %d, want 42", userID)
			}
			return &domain.UserWallet{ID: 7, UserID: userID, Balance: 1000}, nil
		},
	}
	client := newWalletServiceClient(t, stub)

	response, err := client.ProvisionWallet(context.Background(), &walletv1.ProvisionWalletRequest{UserId: 42})
	if err != nil {
		t.Fatalf("ProvisionWallet returned an error: %v", err)
	}
	if response.GetWalletId() != 7 || response.GetUserId() != 42 || response.GetBalance() != 1000 {
		t.Fatalf("unexpected wallet response: %+v", response)
	}
}

func TestProvisionWalletRejectsInvalidRequest(t *testing.T) {
	handler := walletgrpc.NewWalletGRPCHandler(&walletServiceStub{})
	for _, request := range []*walletv1.ProvisionWalletRequest{nil, {}} {
		_, err := handler.ProvisionWallet(context.Background(), request)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("error code = %s, want %s; error=%v", status.Code(err), codes.InvalidArgument, err)
		}
	}
}

func TestProvisionWalletHidesInternalError(t *testing.T) {
	client := newWalletServiceClient(t, &walletServiceStub{
		provisionWallet: func(context.Context, int64) (*domain.UserWallet, error) {
			return nil, errors.New("database password exposed")
		},
	})

	_, err := client.ProvisionWallet(context.Background(), &walletv1.ProvisionWalletRequest{UserId: 42})
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal server error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteWalletUsesAuthenticatedIdentityFromRequest(t *testing.T) {
	deletedUserID := int64(0)
	client := newWalletServiceClient(t, &walletServiceStub{
		deleteWallet: func(_ context.Context, userID int64) error {
			deletedUserID = userID
			return nil
		},
	})

	_, err := client.DeleteWallet(context.Background(), &walletv1.DeleteWalletRequest{UserId: 42})
	if err != nil {
		t.Fatalf("DeleteWallet returned an error: %v", err)
	}
	if deletedUserID != 42 {
		t.Fatalf("deleted user ID = %d, want 42", deletedUserID)
	}
}

func TestGetBalanceMapsServiceResult(t *testing.T) {
	client := newWalletServiceClient(t, &walletServiceStub{
		getBalance: func(_ context.Context, userID int64) (*domain.WalletBalanceResponse, error) {
			return &domain.WalletBalanceResponse{UserID: userID, Balance: 1250}, nil
		},
	})

	response, err := client.GetBalance(context.Background(), &walletv1.GetBalanceRequest{UserId: 42})
	if err != nil {
		t.Fatalf("GetBalance returned an error: %v", err)
	}
	if response.GetUserId() != 42 || response.GetBalance() != 1250 {
		t.Fatalf("unexpected balance response: %+v", response)
	}
}
