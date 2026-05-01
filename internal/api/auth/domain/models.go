package domain

import (
	"encoding/json"
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

// PublicUserProfile is returned by GET /users/{userRef} (no email or verification flags).
type PublicUserProfile struct {
	ID             uuid.UUID             `json:"id"`
	Handle         string                `json:"handle"`
	DisplayName    *string               `json:"display_name,omitempty"`
	AvatarURL      *string               `json:"avatar_url,omitempty"`
	Status         UserStatus            `json:"status"`
	CreatedAt      time.Time             `json:"created_at"`
	SolvedProblems []PublicSolvedProblem `json:"solved_problems"`
}

type PublicSolvedProblem struct {
	ID         uuid.UUID      `json:"id"`
	Slug       string         `json:"slug"`
	Title      string         `json:"title"`
	Summary    string         `json:"summary"`
	Difficulty string         `json:"difficulty"`
	Solution   PublicSolution `json:"solution"`
	SolvedAt   time.Time      `json:"solved_at"`
}

type PublicSolution struct {
	ID                  uuid.UUID `json:"id"`
	LanguageID          uuid.UUID `json:"language_id"`
	LanguageKey         string    `json:"language_key"`
	LanguageDisplayName string    `json:"language_display_name"`
	SourceText          *string   `json:"source_text,omitempty"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

type AuthIdentity struct {
	ID                      uuid.UUID  `json:"id"`
	UserID                  uuid.UUID  `json:"user_id"`
	Provider                string     `json:"provider"`
	ProviderSubject         string     `json:"provider_subject"`
	PasswordHash            *string    `json:"-"`
	PasswordAlgo            *string    `json:"-"`
	EmailAtProvider         *string    `json:"email_at_provider,omitempty"`
	EmailVerifiedAtProvider *bool      `json:"email_verified_at_provider,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	LastLoginAt             *time.Time `json:"last_login_at,omitempty"`
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

type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte
	CreatedAt time.Time
}

type UserStats struct {
	TotalSolved      int                    `json:"total_solved"`
	SolvedByDifficulty map[string]int        `json:"solved_by_difficulty"`
	SolvedByTag      []TagStat              `json:"solved_by_tag"`
	RecentSubmissions []RecentSubmission     `json:"recent_submissions"`
	SubmissionStats  SubmissionStats        `json:"submission_stats"`
}

type TagStat struct {
	TagID    uuid.UUID `json:"tag_id"`
	TagName  string    `json:"tag_name"`
	Count    int       `json:"count"`
	Problems []int     `json:"problems"`
}

type RecentSubmission struct {
	ProblemID    uuid.UUID `json:"problem_id"`
	ProblemSlug  string    `json:"problem_slug"`
	ProblemTitle string    `json:"problem_title"`
	LanguageKey  string    `json:"language_key"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type SubmissionStats struct {
	TotalSubmissions int      `json:"total_submissions"`
	TotalTestRuns    int      `json:"total_test_runs"`
	AcceptanceRate   float64  `json:"acceptance_rate"`
	MostUsedLanguages []LanguageStat `json:"most_used_languages"`
}

type LanguageStat struct {
	LanguageKey  string `json:"language_key"`
	LanguageName string `json:"language_name"`
	Count        int    `json:"count"`
}

type AdminUserSummary struct {
	ID              uuid.UUID  `json:"id"`
	Handle          string     `json:"handle"`
	Email           *string    `json:"email,omitempty"`
	DisplayName     *string    `json:"display_name,omitempty"`
	Status          UserStatus `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	ViolationCount  int        `json:"violation_count"`
	SubmissionCount int        `json:"submission_count"`
}

type SecurityEvent struct {
	ID           uuid.UUID       `json:"id"`
	SubmissionID *uuid.UUID      `json:"submission_id,omitempty"`
	Category     string          `json:"category"`
	Severity     string          `json:"severity"`
	DetailJSON   json.RawMessage `json:"detail_json"`
	CreatedAt    time.Time       `json:"created_at"`
}
