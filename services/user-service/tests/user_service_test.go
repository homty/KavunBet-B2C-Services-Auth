package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"betwin/user-service/internal/domain"
	"betwin/user-service/internal/repository"
	"betwin/user-service/internal/service"
)

// MockProfileRepository simulates PostgreSQL profile table in-memory.
type MockProfileRepository struct {
	profiles map[int64]*domain.UserProfile
	usernames map[int64]string
}

func NewMockProfileRepository() *MockProfileRepository {
	return &MockProfileRepository{
		profiles:  make(map[int64]*domain.UserProfile),
		usernames: make(map[int64]string),
	}
}

func (m *MockProfileRepository) GetProfileByUserID(ctx context.Context, userID int64) (*domain.UserProfile, error) {
	profile, exists := m.profiles[userID]
	if !exists {
		return nil, repository.ErrProfileNotFound
	}
	return profile, nil
}

func (m *MockProfileRepository) UpdateProfile(ctx context.Context, userID int64, req domain.UpdateProfileRequest) (*domain.UserProfile, error) {
	profile, exists := m.profiles[userID]
	if !exists {
		return nil, repository.ErrProfileNotFound
	}

	if req.ContactInfo != nil {
		profile.ContactInfo = req.ContactInfo
	}
	if req.DisplayName != nil {
		profile.DisplayName = req.DisplayName
	}
	if req.Bio != nil {
		profile.Bio = req.Bio
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = req.AvatarURL
	}
	profile.UpdatedAt = time.Now().UTC()

	return profile, nil
}

func (m *MockProfileRepository) GetPublicProfileByID(ctx context.Context, userID int64) (*domain.PublicProfileResponse, error) {
	profile, exists := m.profiles[userID]
	if !exists {
		return nil, repository.ErrProfileNotFound
	}

	username, ok := m.usernames[userID]
	if !ok {
		username = "unknown_user"
	}

	return &domain.PublicProfileResponse{
		UserID:      profile.UserID,
		Username:    username,
		DisplayName: profile.DisplayName,
		Bio:         profile.Bio,
		AvatarURL:   profile.AvatarURL,
	}, nil
}

func setupTestUserService() (service.UserService, *MockProfileRepository) {
	mockRepo := NewMockProfileRepository()
	svc := service.NewUserService(mockRepo)
	return svc, mockRepo
}

func TestGetMyProfile_Success(t *testing.T) {
	svc, mockRepo := setupTestUserService()
	ctx := context.Background()

	// Seed profile
	contact := "alice@example.com"
	displayName := "Alice Wonder"
	mockRepo.profiles[1] = &domain.UserProfile{
		UserID:      1,
		ContactInfo: &contact,
		DisplayName: &displayName,
		UpdatedAt:   time.Now(),
	}

	profile, err := svc.GetMyProfile(ctx, 1)
	if err != nil {
		t.Fatalf("Expected successful profile fetch, got error: %v", err)
	}

	if profile.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", profile.UserID)
	}
	if profile.DisplayName == nil || *profile.DisplayName != "Alice Wonder" {
		t.Errorf("Expected DisplayName 'Alice Wonder', got %v", profile.DisplayName)
	}
}

func TestGetMyProfile_NotFound(t *testing.T) {
	svc, _ := setupTestUserService()
	ctx := context.Background()

	_, err := svc.GetMyProfile(ctx, 9999)
	if !errors.Is(err, repository.ErrProfileNotFound) {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}
}

func TestUpdateMyProfile_Success(t *testing.T) {
	svc, mockRepo := setupTestUserService()
	ctx := context.Background()

	// Seed profile
	contact := "bob@example.com"
	mockRepo.profiles[2] = &domain.UserProfile{
		UserID:      2,
		ContactInfo: &contact,
		UpdatedAt:   time.Now(),
	}

	newDisplayName := "Bobby Fisher"
	newBio := "Chess champion & prediction trader"
	newAvatar := "https://example.com/avatar.jpg"

	updateReq := domain.UpdateProfileRequest{
		DisplayName: &newDisplayName,
		Bio:         &newBio,
		AvatarURL:   &newAvatar,
	}

	updated, err := svc.UpdateMyProfile(ctx, 2, updateReq)
	if err != nil {
		t.Fatalf("Expected successful profile update, got error: %v", err)
	}

	if updated.DisplayName == nil || *updated.DisplayName != "Bobby Fisher" {
		t.Errorf("Expected DisplayName 'Bobby Fisher', got %v", updated.DisplayName)
	}
	if updated.Bio == nil || *updated.Bio != "Chess champion & prediction trader" {
		t.Errorf("Expected Bio update, got %v", updated.Bio)
	}
}

func TestUpdateMyProfile_ValidationErrors(t *testing.T) {
	svc, mockRepo := setupTestUserService()
	ctx := context.Background()

	mockRepo.profiles[3] = &domain.UserProfile{UserID: 3, UpdatedAt: time.Now()}

	// 1. DisplayName too long (>100 chars)
	longName := string(make([]byte, 101))
	_, err := svc.UpdateMyProfile(ctx, 3, domain.UpdateProfileRequest{DisplayName: &longName})
	if !errors.Is(err, service.ErrDisplayNameTooLong) {
		t.Errorf("Expected ErrDisplayNameTooLong, got %v", err)
	}

	// 2. Bio too long (>500 chars)
	longBio := string(make([]byte, 501))
	_, err = svc.UpdateMyProfile(ctx, 3, domain.UpdateProfileRequest{Bio: &longBio})
	if !errors.Is(err, service.ErrBioTooLong) {
		t.Errorf("Expected ErrBioTooLong, got %v", err)
	}

	// 3. AvatarURL too long (>512 chars)
	longAvatar := string(make([]byte, 513))
	_, err = svc.UpdateMyProfile(ctx, 3, domain.UpdateProfileRequest{AvatarURL: &longAvatar})
	if !errors.Is(err, service.ErrAvatarURLTooLong) {
		t.Errorf("Expected ErrAvatarURLTooLong, got %v", err)
	}
}

func TestGetPublicProfile_Success(t *testing.T) {
	svc, mockRepo := setupTestUserService()
	ctx := context.Background()

	displayName := "Charlie Brown"
	bio := "Peanuts trader"
	mockRepo.profiles[4] = &domain.UserProfile{
		UserID:      4,
		DisplayName: &displayName,
		Bio:         &bio,
		UpdatedAt:   time.Now(),
	}
	mockRepo.usernames[4] = "charlie_b"

	pub, err := svc.GetPublicProfile(ctx, 4)
	if err != nil {
		t.Fatalf("Expected successful public profile fetch, got error: %v", err)
	}

	if pub.UserID != 4 {
		t.Errorf("Expected UserID 4, got %d", pub.UserID)
	}
	if pub.Username != "charlie_b" {
		t.Errorf("Expected Username 'charlie_b', got '%s'", pub.Username)
	}
}
