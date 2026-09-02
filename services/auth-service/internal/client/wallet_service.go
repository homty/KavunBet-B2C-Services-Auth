package client

import (
	"context"
	"time"

	walletv1 "betwin/pkg/proto/wallet/v1"
)

const walletProvisioningTimeout = 3 * time.Second

type UserWalletProvisioner struct {
	client walletv1.WalletServiceClient
}

func NewUserWalletProvisioner(client walletv1.WalletServiceClient) *UserWalletProvisioner {
	return &UserWalletProvisioner{client: client}
}

func (provisioner *UserWalletProvisioner) ProvisionWallet(ctx context.Context, userID int64) error {
	callContext, cancel := context.WithTimeout(ctx, walletProvisioningTimeout)
	defer cancel()

	_, err := provisioner.client.ProvisionWallet(callContext, &walletv1.ProvisionWalletRequest{UserId: userID})
	return err
}

func (provisioner *UserWalletProvisioner) DeleteWallet(ctx context.Context, userID int64) error {
	callContext, cancel := context.WithTimeout(ctx, walletProvisioningTimeout)
	defer cancel()

	_, err := provisioner.client.DeleteWallet(callContext, &walletv1.DeleteWalletRequest{UserId: userID})
	return err
}
