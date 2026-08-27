package grpc

import (
	userv1 "betwin/pkg/proto/user/v1"
	grpcresponse "betwin/user-service/internal/API/grpc/response"
	"betwin/user-service/internal/service"
	"context"
	"strings"
)

type UserGRPCHandler struct {
	userv1.UnimplementedUserServiceServer
	userService service.UserService
}

func NewUserGRPCHandler(userService service.UserService) *UserGRPCHandler {
	return &UserGRPCHandler{userService: userService}
}

func (handler *UserGRPCHandler) ProvisionProfile(ctx context.Context, request *userv1.ProvisionProfileRequest) (*userv1.ProvisionProfileResponse, error) {
	if request == nil {
		return nil, grpcresponse.InvalidArgument("request is required")
	}
	if request.GetUserId() <= 0 {
		return nil, grpcresponse.InvalidArgument("user_id must be greater than zero")
	}
	if strings.TrimSpace(request.GetUsername()) == "" {
		return nil, grpcresponse.InvalidArgument("username is required")
	}

	profile, err := handler.userService.ProvisionProfile(ctx, request.GetUserId(), request.GetUsername(), request.GetContactInfo())
	if err != nil {
		return nil, grpcresponse.FromError(err)
	}
	if profile == nil {
		return nil, grpcresponse.Internal()
	}

	return &userv1.ProvisionProfileResponse{
		UserId:   profile.UserID,
		Username: profile.Username,
	}, nil
}

func (handler *UserGRPCHandler) GetPublicProfile(ctx context.Context, request *userv1.GetPublicProfileRequest) (*userv1.GetPublicProfileResponse, error) {
	if request == nil {
		return nil, grpcresponse.InvalidArgument("request is required")
	}
	if request.GetTargetUserId() <= 0 {
		return nil, grpcresponse.InvalidArgument("target_user_id must be greater than zero")
	}

	profile, err := handler.userService.GetPublicProfile(ctx, request.GetTargetUserId())
	if err != nil {
		return nil, grpcresponse.FromError(err)
	}
	if profile == nil {
		return nil, grpcresponse.Internal()
	}

	response := &userv1.GetPublicProfileResponse{
		UserId:   profile.UserID,
		Username: profile.Username,
	}
	if profile.DisplayName != nil {
		response.DisplayName = *profile.DisplayName
	}
	if profile.AvatarURL != nil {
		response.AvatarUrl = *profile.AvatarURL
	}
	if profile.Bio != nil {
		response.Bio = *profile.Bio
	}

	return response, nil
}
