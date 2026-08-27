package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	grpcresponse "betwin/user-service/internal/API/grpc/response"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestFromError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    codes.Code
		wantMessage string
	}{
		{name: "profile not found", err: repository.ErrProfileNotFound, wantCode: codes.NotFound, wantMessage: "profile not found"},
		{name: "wrapped profile not found", err: fmt.Errorf("lookup: %w", repository.ErrProfileNotFound), wantCode: codes.NotFound, wantMessage: "profile not found"},
		{name: "validation", err: service.ErrBioTooLong, wantCode: codes.InvalidArgument, wantMessage: service.ErrBioTooLong.Error()},
		{name: "canceled", err: context.Canceled, wantCode: codes.Canceled, wantMessage: "request canceled"},
		{name: "deadline", err: context.DeadlineExceeded, wantCode: codes.DeadlineExceeded, wantMessage: "request deadline exceeded"},
		{name: "internal details hidden", err: errors.New("database password exposed"), wantCode: codes.Internal, wantMessage: "internal server error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grpcError := grpcresponse.FromError(test.err)
			if code := status.Code(grpcError); code != test.wantCode {
				t.Fatalf("code = %s, want %s", code, test.wantCode)
			}
			if message := status.Convert(grpcError).Message(); message != test.wantMessage {
				t.Errorf("message = %q, want %q", message, test.wantMessage)
			}
		})
	}
}
