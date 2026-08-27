package grpc

import (
	userv1 "betwin/pkg/proto/user/v1"
	"betwin/user-service/internal/service"
	"context"
)

type UserGRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	userService service.UserService
}

func NewUserGRPCHandler(userService service.UserService) *UserGRPCHandler {
	return &UserGRPCHandler{userService: userService}
}

func (handler *UserGRPCHandler) ProvisionProfile(ctx context.Context, request *userv1.ProvisionProfileRequest) (*userv1.ProvisionProfileResponse, error) {
	_, err := handler.userService.ProvisionProfile(ctx, request.GetUserId(), request.GetUsername(), request.GetContactInfo())
	if err != nil {
		return nil, err
	}

	return &userv1.ProvisionProfileResponse{
		UserId:   request.GetUserId(),
		Username: request.GetUsername(),
	}, nil
}
