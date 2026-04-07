package followup

import "errors"

var (
	ErrNotFound       = errors.New("follow-up not found")
	ErrUnauthorized   = errors.New("unauthorized access to follow-up")
	ErrInvalidStatus  = errors.New("invalid follow-up status")
	ErrInvalidDueDate = errors.New("due date must be in the future")
)
