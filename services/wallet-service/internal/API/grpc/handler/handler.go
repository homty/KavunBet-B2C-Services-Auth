package grpc

import (
	"context"

	walletv1 "betwin/pkg/proto/wallet/v1"
	grpcresponse "betwin/wallet-service/internal/API/grpc/response"
	"betwin/wallet-service/internal/service"
)

type WalletGRPCHandler struct {
	walletv1.UnimplementedWalletServiceServer
	walletService service.WalletService
}

func NewWalletGRPCHandler(walletService service.WalletService) *WalletGRPCHandler {
	return &WalletGRPCHandler{walletService: walletService}
}

func (handler *WalletGRPCHandler) ProvisionWallet(ctx context.Context, request *walletv1.ProvisionWalletRequest) (*walletv1.ProvisionWalletResponse, error) {
	if request == nil {
		return nil, grpcresponse.InvalidArgument("request is required")
	}
	if request.GetUserId() <= 0 {
		return nil, grpcresponse.InvalidArgument("user_id must be greater than zero")
	}

	wallet, err := handler.walletService.ProvisionWallet(ctx, request.GetUserId())
	if err != nil {
		return nil, grpcresponse.FromError(err)
	}
	if wallet == nil {
		return nil, grpcresponse.Internal()
	}

	return &walletv1.ProvisionWalletResponse{
		WalletId: wallet.ID,
		UserId:   wallet.UserID,
		Balance:  wallet.Balance,
	}, nil
}

func (handler *WalletGRPCHandler) GetBalance(ctx context.Context, request *walletv1.GetBalanceRequest) (*walletv1.GetBalanceResponse, error) {
	if request == nil {
		return nil, grpcresponse.InvalidArgument("request is required")
	}
	if request.GetUserId() <= 0 {
		return nil, grpcresponse.InvalidArgument("user_id must be greater than zero")
	}

	balance, err := handler.walletService.GetBalance(ctx, request.GetUserId())
	if err != nil {
		return nil, grpcresponse.FromError(err)
	}
	if balance == nil {
		return nil, grpcresponse.Internal()
	}

	return &walletv1.GetBalanceResponse{
		UserId:  balance.UserID,
		Balance: balance.Balance,
	}, nil
}

func (handler *WalletGRPCHandler) DeleteWallet(ctx context.Context, request *walletv1.DeleteWalletRequest) (*walletv1.DeleteWalletResponse, error) {
	if request == nil {
		return nil, grpcresponse.InvalidArgument("request is required")
	}
	if request.GetUserId() <= 0 {
		return nil, grpcresponse.InvalidArgument("user_id must be greater than zero")
	}

	if err := handler.walletService.DeleteWallet(ctx, request.GetUserId()); err != nil {
		return nil, grpcresponse.FromError(err)
	}

	return &walletv1.DeleteWalletResponse{}, nil
}
