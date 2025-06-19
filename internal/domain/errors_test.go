package domain

import (
	"errors"
	"testing"
)

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrMessageNotFound",
			err:      ErrMessageNotFound,
			expected: "message not found",
		},
		{
			name:     "ErrSchedulerRunning",
			err:      ErrSchedulerRunning,
			expected: "scheduler is already running",
		},
		{
			name:     "ErrSchedulerNotRunning",
			err:      ErrSchedulerNotRunning,
			expected: "scheduler is not running",
		},
		{
			name:     "ErrMessageTooLong",
			err:      ErrMessageTooLong,
			expected: "message content exceeds maximum length",
		},
		{
			name:     "ErrInvalidPhoneNumber",
			err:      ErrInvalidPhoneNumber,
			expected: "invalid phone number format",
		},
		{
			name:     "ErrWebhookFailed",
			err:      ErrWebhookFailed,
			expected: "webhook request failed",
		},
		{
			name:     "ErrMaxRetriesExceeded",
			err:      ErrMaxRetriesExceeded,
			expected: "maximum retry attempts exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorComparison(t *testing.T) {
	// Test that errors can be compared using errors.Is
	testErr := ErrMessageNotFound

	if !errors.Is(testErr, ErrMessageNotFound) {
		t.Error("Expected errors.Is to return true for same error")
	}

	if errors.Is(testErr, ErrSchedulerRunning) {
		t.Error("Expected errors.Is to return false for different error")
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that domain errors can be wrapped and unwrapped
	wrappedErr := errors.Join(ErrMessageTooLong, errors.New("additional context"))

	if !errors.Is(wrappedErr, ErrMessageTooLong) {
		t.Error("Expected wrapped error to contain domain error")
	}
}

func TestErrorsAreNotNil(t *testing.T) {
	// Ensure all domain errors are properly initialized
	domainErrors := []error{
		ErrMessageNotFound,
		ErrSchedulerRunning,
		ErrSchedulerNotRunning,
		ErrMessageTooLong,
		ErrInvalidPhoneNumber,
		ErrWebhookFailed,
		ErrMaxRetriesExceeded,
	}

	for i, err := range domainErrors {
		if err == nil {
			t.Errorf("Domain error at index %d is nil", i)
		}
	}
}
