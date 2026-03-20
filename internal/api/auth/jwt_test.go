package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	secret := "test-secret"
	accessDuration := 15 * time.Minute
	refreshDuration := 7 * 24 * time.Hour

	manager := NewJWTManager(secret, accessDuration, refreshDuration)

	assert.NotNil(t, manager)
	assert.Equal(t, []byte(secret), manager.secretKey)
	assert.Equal(t, accessDuration, manager.accessTokenDuration)
	assert.Equal(t, refreshDuration, manager.refreshTokenDuration)
}

func TestGenerateAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"test.permission", "another.permission"}

	token, err := manager.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should have 3 parts (header.payload.signature)
	parts := len(token)
	assert.Greater(t, parts, 0)
}

func TestValidateAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"test.permission", "another.permission"}

	token, err := manager.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, handle, claims.Handle)
		assert.Equal(t, permissions, claims.Permissions)
		assert.True(t, claims.ExpiresAt.After(time.Now()))
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("invalid.token.here")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		wrongManager := NewJWTManager("wrong-secret", 15*time.Minute, 7*24*time.Hour)
		_, err := wrongManager.ValidateAccessToken(token)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestValidateAccessToken_Expired(t *testing.T) {
	// Create manager with very short expiration
	manager := NewJWTManager("test-secret", 1*time.Millisecond, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"test.permission"}

	token, err := manager.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = manager.ValidateAccessToken(token)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Run("generates non-empty token", func(t *testing.T) {
		token, err := GenerateRefreshToken()
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err := GenerateRefreshToken()
		require.NoError(t, err)

		token2, err := GenerateRefreshToken()
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("generates base64 URL encoded tokens", func(t *testing.T) {
		token, err := GenerateRefreshToken()
		require.NoError(t, err)

		// Should be valid base64 URL encoding
		assert.NotContains(t, token, "+")
		assert.NotContains(t, token, "/")
	})
}

func TestGetRefreshTokenDuration(t *testing.T) {
	refreshDuration := 7 * 24 * time.Hour
	manager := NewJWTManager("test-secret", 15*time.Minute, refreshDuration)

	assert.Equal(t, refreshDuration, manager.GetRefreshTokenDuration())
}

func TestTokenClaims(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()
	handle := "testuser"
	permissions := []string{"perm1", "perm2", "perm3"}

	token, err := manager.GenerateAccessToken(userID, handle, permissions)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)
	require.NoError(t, err)

	// Verify all claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, handle, claims.Handle)
	assert.Equal(t, permissions, claims.Permissions)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)

	// Verify time relationships
	now := time.Now()
	assert.True(t, claims.IssuedAt.Before(now.Add(time.Second)))
	assert.True(t, claims.NotBefore.Before(now.Add(time.Second)))
	assert.True(t, claims.ExpiresAt.After(now))
}
