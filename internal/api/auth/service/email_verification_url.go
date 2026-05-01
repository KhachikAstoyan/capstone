package service

import (
	"net/url"
	"strings"
)

// buildEmailVerificationURL returns an absolute verification link for the SPA.
// frontendBase should be an origin like https://app.example.com or http://localhost:5173 (trailing slash optional).
// Returns empty string if frontendBase is blank.
func buildEmailVerificationURL(frontendBase, plaintextToken string) string {
	base := strings.TrimSpace(strings.TrimRight(frontendBase, "/"))
	if base == "" || plaintextToken == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return base + "/?token=" + url.QueryEscape(plaintextToken)
	}
	q := u.Query()
	q.Set("token", plaintextToken)
	u.RawQuery = q.Encode()
	return u.String()
}
