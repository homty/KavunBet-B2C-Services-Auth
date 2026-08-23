package service

import (
	"betwin/user-service/internal/domain"
	"betwin/user-service/internal/repository"
	"context"
	"errors"
	"fmt"
)

var (
	ErrDisplayNameTooLong = errors.New("Display name cannot exceed 100 characters")
	ErrBioTooLong         = errors.New("Bio cannot exceed 255 characters")
	ErrAvatarURLTooLong   = errors.New("Avatar URL cannot exceed 512 characters")
)

type UserService interface {
	GetMyProfile(context context.Context, userID int64) (*domain.UserProfile, error)
	UpdateMyProfile(context context.Context, userID int64, request domain.UpdateProfileRequest) (*domain.UserProfile, error)
	GetPublicProfile(context context.Context, targetUserID int64) (*domain.PublicProfileResponse, error)
}

type userService struct {
	profileRepository repository.ProfileRepository
}

func NewUserService(profileRepository repository.ProfileRepository) UserService {
	return &userService{
		profileRepository: profileRepository,
	}
}

func (self *userService) GetMyProfile(context context.Context, userID int64) (*domain.UserProfile, error) {
	profile, err := self.profileRepository.GetProfileByUserID(context, userID)

	if err != nil {
		return nil, err
	}

	profile.UpdatedAt = profile.UpdatedAt.UTC()
	return profile, nil
}

func (self *userService) UpdateMyProfile(context context.Context, userID int64, request domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	if request.DisplayName != nil && len(*request.DisplayName) > 100 {
		return nil, ErrDisplayNameTooLong
	} else if request.Bio != nil && len(*request.Bio) > 255 {
		return nil, ErrBioTooLong
	} else if request.AvatarURL != nil && len(*request.AvatarURL) > 512 {
		return nil, ErrAvatarURLTooLong
	}

	updatedProfile, err := self.profileRepository.UpdateProfile(context, userID, request)
	if err != nil {
		return nil, fmt.Errorf("Failed to update Profile: %w", err)
	}

	updatedProfile.UpdatedAt = updatedProfile.UpdatedAt.UTC()
	return updatedProfile, nil
}

func (self *userService) GetPublicProfile(context context.Context, targetUserID int64) (*domain.PublicProfileResponse, error) {
	return self.profileRepository.GetPublicProfileByID(context, targetUserID)
}
