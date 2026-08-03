package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetProfile(ctx context.Context, userID uuid.UUID) (Profile, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(Profile), args.Error(1)
}

func (m *mockRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Profile, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(Profile), args.Error(1)
}

func TestService_GetProfile_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	expected := Profile{UserID: userID, Email: "player@example.com", Role: "user", Theme: "dark", Language: "ru"}
	repo.On("GetProfile", mock.Anything, userID).Return(expected, nil)

	profile, err := svc.GetProfile(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, expected, profile)
}

func TestService_GetProfile_NotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	repo.On("GetProfile", mock.Anything, userID).Return(Profile{}, ErrProfileNotFound)

	_, err := svc.GetProfile(context.Background(), userID)

	require.Error(t, err)
}

func TestService_UpdateProfile_PassesInputThrough(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)

	userID := uuid.New()
	name := "Nova"
	req := UpdateProfileRequest{DisplayName: &name, Hobbies: []string{"chess", "hiking"}}
	expected := Profile{UserID: userID, DisplayName: &name, Hobbies: []string{"chess", "hiking"}}

	repo.On("UpdateProfile", mock.Anything, userID, UpdateProfileInput(req)).Return(expected, nil)

	profile, err := svc.UpdateProfile(context.Background(), userID, req)

	require.NoError(t, err)
	assert.Equal(t, expected, profile)
	repo.AssertExpectations(t)
}
