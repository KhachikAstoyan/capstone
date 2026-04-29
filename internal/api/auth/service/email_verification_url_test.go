package service

import "testing"

func TestBuildEmailVerificationURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		base   string
		token  string
		expect string
	}{
		{"empty base", "", "abc", ""},
		{"empty token", "http://localhost:5173", "", ""},
		{"origin no slash", "http://localhost:5173", "tok", "http://localhost:5173?token=tok"},
		{"origin with slash", "http://localhost:5173/", "tok", "http://localhost:5173?token=tok"},
		{"preserves path", "https://app.example.com/app/", "x", "https://app.example.com/app?token=x"},
		{"merges query", "https://x.com?foo=1", "t", "https://x.com?foo=1&token=t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildEmailVerificationURL(tt.base, tt.token)
			if got != tt.expect {
				t.Fatalf("buildEmailVerificationURL(%q, %q) = %q, want %q", tt.base, tt.token, got, tt.expect)
			}
		})
	}
}
