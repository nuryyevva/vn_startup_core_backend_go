package auth

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type bonusSpy struct{ granted map[uuid.UUID]int32 }

func (b *bonusSpy) GrantSignupBonus(_ context.Context, userID uuid.UUID, amount int32) error {
	b.granted[userID] += amount
	return nil
}

func encodedHash(t *testing.T, password string) (hash, encoded string) {
	t.Helper()
	hash, err := HashPassword(password)
	require.NoError(t, err)
	return hash, base64.StdEncoding.EncodeToString([]byte(hash))
}

func TestParseAdminAccounts(t *testing.T) {
	hash, encoded := encodedHash(t, "correct horse battery")

	accounts, err := ParseAdminAccounts(" a@example.com:" + encoded + " ,, b@example.com:" + encoded)
	require.NoError(t, err)
	assert.Equal(t, []AdminAccount{{Email: "a@example.com", PasswordHash: hash}, {Email: "b@example.com", PasswordHash: hash}}, accounts)

	accounts, err = ParseAdminAccounts("")
	require.NoError(t, err)
	assert.Empty(t, accounts)

	for _, bad := range []string{
		"a@example.com",           // no hash
		"not-an-email:" + encoded, // bad email
		"a@example.com:%%%",       // not base64
		"a@example.com:" + base64.StdEncoding.EncodeToString([]byte("plain password")), // not a hash
	} {
		_, err := ParseAdminAccounts(bad)
		assert.Error(t, err, bad)
	}
}

func TestEnsureAdmins_CreatesMissingAndPromotesExisting(t *testing.T) {
	repo := new(mockRepository)
	bonus := &bonusSpy{granted: map[uuid.UUID]int32{}}
	svc := NewService(repo, newTestIssuer()).WithSignupBonus(bonus, 600)

	hash, _ := encodedHash(t, "correct horse battery")
	existing := User{ID: uuid.New(), Email: "old@example.com", Role: "user"}
	created := User{ID: uuid.New(), Email: "new@example.com", Role: "user"}

	repo.On("GetUserByEmail", mock.Anything, "old@example.com").Return(existing, nil)
	repo.On("GetUserByEmail", mock.Anything, "new@example.com").Return(User{}, ErrUserNotFound)
	repo.On("CreateUser", mock.Anything, "new@example.com", hash).Return(created, nil)
	repo.On("MakeAdmin", mock.Anything, existing.ID, hash).Return(nil)
	repo.On("MakeAdmin", mock.Anything, created.ID, hash).Return(nil)

	err := svc.EnsureAdmins(context.Background(), []AdminAccount{
		{Email: "old@example.com", PasswordHash: hash},
		{Email: "new@example.com", PasswordHash: hash},
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
	assert.Equal(t, map[uuid.UUID]int32{created.ID: 600}, bonus.granted, "only a newly created admin gets the starting diamonds")
}
