package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  nil,
		},
		{
			name:     "minimum length password",
			password: "12345678",
			wantErr:  nil,
		},
		{
			name:     "password too short",
			password: "1234567",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "password too long",
			password: strings.Repeat("a", 129),
			wantErr:  ErrPasswordTooLong,
		},
		{
			name:     "maximum length password",
			password: strings.Repeat("a", 128),
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHashAndValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "password too short",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "password too long",
			password: strings.Repeat("a", 129),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashAndValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash)
				assert.True(t, strings.HasPrefix(hash, "$2a$"))
			}
		})
	}
}

func TestComparePassword(t *testing.T) {
	password := "mySecurePassword123"
	hash, err := HashAndValidatePassword(password)
	require.NoError(t, err)

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  error
	}{
		{
			name:     "correct password",
			hash:     hash,
			password: password,
			wantErr:  nil,
		},
		{
			name:     "incorrect password",
			hash:     hash,
			password: "wrongPassword",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "empty password",
			hash:     hash,
			password: "",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "invalid hash format",
			hash:     "not-a-valid-hash",
			password: password,
			wantErr:  ErrInvalidPassword, // bcrypt error gets wrapped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ComparePassword(tt.hash, tt.password)
			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHashPasswordDeterminism(t *testing.T) {
	password := "testPassword123"
	
	hash1, err := HashAndValidatePassword(password)
	require.NoError(t, err)
	
	hash2, err := HashAndValidatePassword(password)
	require.NoError(t, err)
	
	// Hashes should be different due to salt
	assert.NotEqual(t, hash1, hash2)
	
	// But both should validate correctly
	assert.NoError(t, ComparePassword(hash1, password))
	assert.NoError(t, ComparePassword(hash2, password))
}
