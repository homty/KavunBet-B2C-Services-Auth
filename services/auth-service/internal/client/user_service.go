package client

import (
	"context"

	userv1 "betwin/pkg/proto/user/v1"
)

type UserProfileProvisioner struct {
	client userv1.UserServiceClient
}

func NewUserProfileProvisioner(client userv1.UserServiceClient) *UserProfileProvisioner {
	return &UserProfileProvisioner{client: client}
}

func (provisioner *UserProfileProvisioner) ProvisionProfile(ctx context.Context, userID int64, username, contactInfo string) error {
	_, err := provisioner.client.ProvisionProfile(ctx, &userv1.ProvisionProfileRequest{
		UserId:      userID,
		Username:    username,
		ContactInfo: contactInfo,
	})
	return err
}
