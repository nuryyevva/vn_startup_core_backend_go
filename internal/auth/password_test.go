package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	require.NoError(t, err)
	assert.NotEqual(t, "correct-horse-battery-staple", hash)

	ok, err := VerifyPassword("correct-horse-battery-staple", hash)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = VerifyPassword("wrong-password", hash)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHashPassword_ProducesUniqueSalts(t *testing.T) {
	hash1, err := HashPassword("same-password")
	require.NoError(t, err)
	hash2, err := HashPassword("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
}
