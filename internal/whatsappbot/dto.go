package whatsappbot

import "time"

// InboundMessage represents a standardized message received from a external provider.
type InboundMessage struct {
	From        string
	Body        string
	ProviderMsgID string
	ReceivedAt  time.Time
}

type OutboundReply struct {
	Body string
}

type WhatsAppMessageDTO struct {
	ID                string    `json:"id"`
	Direction         string    `json:"direction"`
	PhoneNumber       string    `json:"phone_number"`
	MessageType       string    `json:"message_type"`
	Content           string    `json:"content"`
	ProviderMessageID *string   `json:"provider_message_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type WhatsAppBotStatusDTO struct {
	IsReady         bool       `json:"is_ready"`
	PhoneNumber     *string    `json:"phone_number"`
	LastInteraction *time.Time `json:"last_interaction"`
	OptInStatus     bool       `json:"opt_in_status"`
}
