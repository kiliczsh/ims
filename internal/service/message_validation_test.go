package service

import (
	"testing"
)

func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name        string
		phoneNumber string
		expected    bool
	}{
		{
			name:        "Valid US number",
			phoneNumber: "+1234567890",
			expected:    true,
		},
		{
			name:        "Valid international number",
			phoneNumber: "+447123456789",
			expected:    true,
		},
		{
			name:        "Valid with spaces (trimmed)",
			phoneNumber: " +1234567890 ",
			expected:    true,
		},
		{
			name:        "Invalid - no plus sign",
			phoneNumber: "1234567890",
			expected:    false,
		},
		{
			name:        "Invalid - starts with zero",
			phoneNumber: "+0123456789",
			expected:    false,
		},
		{
			name:        "Invalid - contains letters",
			phoneNumber: "+123abc4567",
			expected:    false,
		},
		{
			name:        "Invalid - empty string",
			phoneNumber: "",
			expected:    false,
		},
		{
			name:        "Invalid - only plus sign",
			phoneNumber: "+",
			expected:    false,
		},
		{
			name:        "Invalid - too short",
			phoneNumber: "+1",
			expected:    false,
		},
		{
			name:        "Valid - minimum length",
			phoneNumber: "+12",
			expected:    true,
		},
		{
			name:        "Valid - maximum length (15 digits)",
			phoneNumber: "+123456789012345",
			expected:    true,
		},
		{
			name:        "Invalid - too long (16 digits)",
			phoneNumber: "+1234567890123456",
			expected:    false,
		},
		{
			name:        "Invalid - contains spaces inside",
			phoneNumber: "+123 456 7890",
			expected:    false,
		},
		{
			name:        "Invalid - contains dashes",
			phoneNumber: "+123-456-7890",
			expected:    false,
		},
		{
			name:        "Invalid - contains parentheses",
			phoneNumber: "+1(234)567890",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePhoneNumber(tt.phoneNumber)
			if result != tt.expected {
				t.Errorf("validatePhoneNumber(%q) = %v, expected %v", tt.phoneNumber, result, tt.expected)
			}
		})
	}
}
