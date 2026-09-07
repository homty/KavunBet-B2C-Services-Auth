package client

import (
	"context"
	"time"

	userv1 "betwin/auth-service/internal/contracts/user/v1"
)

const provisioningTimeout = 3 * time.Second

type UserProfileProvisioner struct {
	client userv1.UserServiceClient
}

func NewUserProfileProvisioner(client userv1.UserServiceClient) *UserProfileProvisioner {
	return &UserProfileProvisioner{client: client}
}

func (provisioner *UserProfileProvisioner) ProvisionProfile(ctx context.Context, userID int64, username, contactInfo string) error {
	callContext, cancel := context.WithTimeout(ctx, provisioningTimeout)
	defer cancel()

	_, err := provisioner.client.ProvisionProfile(callContext, &userv1.ProvisionProfileRequest{
		UserId:      userID,
		Username:    username,
		ContactInfo: contactInfo,
	})
	return err
}

func (provisioner *UserProfileProvisioner) DeleteProfile(ctx context.Context, userID int64) error {
	callContext, cancel := context.WithTimeout(ctx, provisioningTimeout)
	defer cancel()

	_, err := provisioner.client.DeleteProfile(callContext, &userv1.DeleteProfileRequest{UserId: userID})
	return err
}
