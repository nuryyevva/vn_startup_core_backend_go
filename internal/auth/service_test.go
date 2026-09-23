package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/pkg/apperr"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	args := m.Called(ctx, email, passwordHash)
	return args.Get(0).(User), args.Error(1)
}

func (m *mockRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(User), args.Error(1)
}

func (m *mockRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(User), args.Error(1)
}

func (m *mockRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	args := m.Called(ctx, id, passwordHash)
	return args.Error(0)
}

func newTestIssuer() *TokenIssuer {
	return NewTokenIssuer("test-secret-value-1234567890", 15*time.Minute, 720*time.Hour)
}

func TestService_Register_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	created := User{ID: uuid.New(), Email: "player@example.com", Role: "user", PasswordHash: "irrelevant"}
	repo.On("CreateUser", mock.Anything, "player@example.com", mock.AnythingOfType("string")).Return(created, nil)

	user, tokens, err := svc.Register(context.Background(), RegisterRequest{Email: "player@example.com", Password: "password123"})

	require.NoError(t, err)
	assert.Equal(t, created.ID, user.ID)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	repo.AssertExpectations(t)
}

func TestService_Register_InvalidEmail(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	_, _, err := svc.Register(context.Background(), RegisterRequest{Email: "not-an-email", Password: "password123"})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_email", appErr.Code)
	repo.AssertNotCalled(t, "CreateUser")
}

func TestService_Register_WeakPassword(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	_, _, err := svc.Register(context.Background(), RegisterRequest{Email: "player@example.com", Password: "short"})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "weak_password", appErr.Code)
}

func TestService_Register_EmailTaken(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	repo.On("CreateUser", mock.Anything, "player@example.com", mock.AnythingOfType("string")).Return(User{}, ErrEmailTaken)

	_, _, err := svc.Register(context.Background(), RegisterRequest{Email: "player@example.com", Password: "password123"})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "email_taken", appErr.Code)
}

func TestService_Login_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	hash, err := HashPassword("password123")
	require.NoError(t, err)
	existing := User{ID: uuid.New(), Email: "player@example.com", Role: "user", PasswordHash: hash}

	repo.On("GetUserByEmail", mock.Anything, "player@example.com").Return(existing, nil)

	user, tokens, err := svc.Login(context.Background(), LoginRequest{Email: "player@example.com", Password: "password123"})

	require.NoError(t, err)
	assert.Equal(t, existing.ID, user.ID)
	assert.NotEmpty(t, tokens.AccessToken)
}

func TestService_Login_UserNotFound(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	repo.On("GetUserByEmail", mock.Anything, "ghost@example.com").Return(User{}, ErrUserNotFound)

	_, _, err := svc.Login(context.Background(), LoginRequest{Email: "ghost@example.com", Password: "password123"})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_credentials", appErr.Code)
}

func TestService_Login_WrongPassword(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	hash, err := HashPassword("correct-password")
	require.NoError(t, err)
	existing := User{ID: uuid.New(), Email: "player@example.com", Role: "user", PasswordHash: hash}

	repo.On("GetUserByEmail", mock.Anything, "player@example.com").Return(existing, nil)

	_, _, err = svc.Login(context.Background(), LoginRequest{Email: "player@example.com", Password: "wrong-password"})

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_credentials", appErr.Code)
}

func TestService_Refresh_Success(t *testing.T) {
	repo := new(mockRepository)
	issuer := newTestIssuer()
	svc := NewService(repo, issuer)

	user := User{ID: uuid.New(), Email: "player@example.com", Role: "user"}
	refreshToken, err := issuer.GenerateRefreshToken(user.ID, user.Role)
	require.NoError(t, err)

	repo.On("GetUserByID", mock.Anything, user.ID).Return(user, nil)

	tokens, err := svc.Refresh(context.Background(), refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
}

func TestService_Refresh_RejectsAccessToken(t *testing.T) {
	repo := new(mockRepository)
	issuer := newTestIssuer()
	svc := NewService(repo, issuer)

	user := User{ID: uuid.New(), Email: "player@example.com", Role: "user"}
	accessToken, err := issuer.GenerateAccessToken(user.ID, user.Role)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), accessToken)

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_refresh_token", appErr.Code)
}

func TestService_ChangePassword_Success(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	userID := uuid.New()
	hash, err := HashPassword("old-password")
	require.NoError(t, err)
	user := User{ID: userID, PasswordHash: hash}

	repo.On("GetUserByID", mock.Anything, userID).Return(user, nil)
	repo.On("UpdatePasswordHash", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil)

	err = svc.ChangePassword(context.Background(), userID, "old-password", "new-password")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	userID := uuid.New()
	hash, err := HashPassword("old-password")
	require.NoError(t, err)
	user := User{ID: userID, PasswordHash: hash}

	repo.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	err = svc.ChangePassword(context.Background(), userID, "wrong-password", "new-password")

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "invalid_credentials", appErr.Code)
	repo.AssertNotCalled(t, "UpdatePasswordHash", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_ChangePassword_WeakNewPassword(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo, newTestIssuer())

	userID := uuid.New()
	hash, err := HashPassword("old-password")
	require.NoError(t, err)
	user := User{ID: userID, PasswordHash: hash}

	repo.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	err = svc.ChangePassword(context.Background(), userID, "old-password", "short")

	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "weak_password", appErr.Code)
	repo.AssertNotCalled(t, "UpdatePasswordHash", mock.Anything, mock.Anything, mock.Anything)
}
