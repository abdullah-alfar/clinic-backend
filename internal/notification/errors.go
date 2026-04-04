package notification

import "errors"

var (
	ErrPreferenceNotFound = errors.New("notification preferences not found")
	ErrInvalidChannel     = errors.New("unsupported notification channel")
	ErrInvalidEvent       = errors.New("unsupported notification event type")
	ErrNoRecipient        = errors.New("patient has no valid recipient address for this channel")
)
