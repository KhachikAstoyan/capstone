package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmth(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "positive number",
			input:    5,
			expected: 10,
		},
		{
			name:     "zero",
			input:    0,
			expected: 0,
		},
		{
			name:     "negative number",
			input:    -5,
			expected: 0,
		},
		{
			name:     "large positive number",
			input:    1000,
			expected: 2000,
		},
		{
			name:     "negative number close to zero",
			input:    -1,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Smth(tt.input)
			assert.Equal(t, tt.expected, result, "Smth(%d) should return %d", tt.input, tt.expected)
		})
	}
}
