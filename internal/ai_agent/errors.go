package ai_agent

import "errors"

var (
	ErrActionNotFound    = errors.New("pending action not found or expired")
	ErrActionAlreadyDone = errors.New("action already confirmed and executed")
	ErrToolExecution     = errors.New("failed to execute tool")
	ErrInvalidPayload    = errors.New("invalid action payload")
	ErrUnauthorized      = errors.New("unauthorized tool execution")
)
