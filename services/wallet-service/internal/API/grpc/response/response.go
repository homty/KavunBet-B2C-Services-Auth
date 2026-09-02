package response

import (
	"context"
	"errors"

	"betwin/wallet-service/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const internalServerErrorMessage = "internal server error"

func InvalidArgument(message string) error {
	return status.Error(codes.InvalidArgument, message)
}

func Internal() error {
	return status.Error(codes.Internal, internalServerErrorMessage)
}

func FromError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, repository.ErrWalletNotFound):
		return status.Error(codes.NotFound, "wallet not found")
	default:
		return Internal()
	}
}
