package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nonsoemeka-backend/internal/auth"
)

func TestAuth_PasswordHashing(t *testing.T) {
	password := "SecretPass123!"
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, auth.CheckPassword(password, hash))
	assert.False(t, auth.CheckPassword("WrongPass", hash))
}

func TestAuth_JWTTokens(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key-min-32-chars-long"

	tokenStr, err := auth.GenerateAccessToken(userID, "admin_user", "ADMIN", secret, 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	claims, err := auth.ParseAccessToken(tokenStr, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "admin_user", claims.Username)
	assert.Equal(t, "ADMIN", claims.Role)
}

func TestAuth_RefreshTokenHash(t *testing.T) {
	rawToken := auth.GenerateRawRefreshToken()
	assert.NotEmpty(t, rawToken)

	hash1 := auth.HashRefreshToken(rawToken)
	hash2 := auth.HashRefreshToken(rawToken)
	assert.Equal(t, hash1, hash2)
}
