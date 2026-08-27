package response

import (
	"context"
	"errors"

	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const internalServerErrorMessage = "internal server error"

// InvalidArgument creates a client-safe validation error.
func InvalidArgument(message string) error {
	return status.Error(codes.InvalidArgument, message)
}

// Internal creates an error that does not expose implementation details.
func Internal() error {
	return status.Error(codes.Internal, internalServerErrorMessage)
}

// FromError translates service and repository errors into client-safe gRPC statuses.
func FromError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, repository.ErrProfileNotFound):
		return status.Error(codes.NotFound, "profile not found")
	case errors.Is(err, service.ErrDisplayNameTooLong),
		errors.Is(err, service.ErrBioTooLong),
		errors.Is(err, service.ErrAvatarURLTooLong):
		return InvalidArgument(err.Error())
	default:
		return Internal()
	}
}
