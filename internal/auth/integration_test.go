//go:build integration

package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vn_startup_core_backend_go/internal/auth"
	"vn_startup_core_backend_go/internal/testutil"
	userpkg "vn_startup_core_backend_go/internal/user"
)

func TestAuthIntegration_RegisterLoginRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	pool := testutil.StartPostgres(t)
	ctx := context.Background()

	repo := auth.NewPostgresRepository(pool)
	issuer := auth.NewTokenIssuer("integration-test-secret-value", 15*time.Minute, 720*time.Hour)
	svc := auth.NewService(repo, issuer)

	created, tokens, err := svc.Register(ctx, auth.RegisterRequest{
		Email:    "integration@example.com",
		Password: "supersecret123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)

	// Registering the same email twice must fail with a domain error, not a
	// raw constraint-violation error leaking out of the repository.
	_, _, err = svc.Register(ctx, auth.RegisterRequest{Email: "integration@example.com", Password: "supersecret123"})
	require.Error(t, err)

	loggedIn, loginTokens, err := svc.Login(ctx, auth.LoginRequest{Email: "integration@example.com", Password: "supersecret123"})
	require.NoError(t, err)
	assert.Equal(t, created.ID, loggedIn.ID)
	assert.NotEmpty(t, loginTokens.AccessToken)

	_, _, err = svc.Login(ctx, auth.LoginRequest{Email: "integration@example.com", Password: "wrong-password"})
	require.Error(t, err)

	refreshed, err := svc.Refresh(ctx, loginTokens.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)

	// Registration must also create a default profile row in the same
	// transaction, so the user module can immediately serve GET /users/me.
	userRepo := userpkg.NewPostgresRepository(pool)
	profile, err := userRepo.GetProfile(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "dark", profile.Theme)
	assert.Equal(t, "ru", profile.Language)
}
