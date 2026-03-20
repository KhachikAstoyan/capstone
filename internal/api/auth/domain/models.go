package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive UserStatus = "ACTIVE"
	UserStatusBanned UserStatus = "BANNED"
)

type User struct {
	ID            uuid.UUID  `json:"id"`
	Handle        string     `json:"handle"`
	Email         *string    `json:"email,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	DisplayName   *string    `json:"display_name,omitempty"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	Status        UserStatus `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AuthIdentity struct {
	ID                       uuid.UUID  `json:"id"`
	UserID                   uuid.UUID  `json:"user_id"`
	Provider                 string     `json:"provider"`
	ProviderSubject          string     `json:"provider_subject"`
	PasswordHash             *string    `json:"-"`
	PasswordAlgo             *string    `json:"-"`
	EmailAtProvider          *string    `json:"email_at_provider,omitempty"`
	EmailVerifiedAtProvider  *bool      `json:"email_verified_at_provider,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	LastLoginAt              *time.Time `json:"last_login_at,omitempty"`
}

type RefreshToken struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	AuthIdentityID *uuid.UUID `json:"auth_identity_id,omitempty"`
	TokenHash      []byte     `json:"-"`
	IssuedAt       time.Time  `json:"issued_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	ReplacedBy     *uuid.UUID `json:"replaced_by,omitempty"`
}
