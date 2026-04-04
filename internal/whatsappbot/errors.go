package whatsappbot

import "errors"

var (
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrSessionNotFound         = errors.New("bot session not found")
	ErrPatientNotLinked        = errors.New("phone number is not linked to a patient")
)
