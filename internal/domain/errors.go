package domain

import "errors"

var (
	ErrMessageNotFound     = errors.New("message not found")
	ErrSchedulerRunning    = errors.New("scheduler is already running")
	ErrSchedulerNotRunning = errors.New("scheduler is not running")
	ErrMessageTooLong      = errors.New("message content exceeds maximum length")
	ErrInvalidPhoneNumber  = errors.New("invalid phone number format")
	ErrWebhookFailed       = errors.New("webhook request failed")
	ErrMaxRetriesExceeded  = errors.New("maximum retry attempts exceeded")
)
