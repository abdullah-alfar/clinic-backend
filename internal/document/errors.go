package document

import "errors"

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrUnauthorized     = errors.New("unauthorized access to document")
	ErrInvalidInput     = errors.New("invalid input")
	ErrStorageFailed    = errors.New("failed to save/delete document in storage")
)
