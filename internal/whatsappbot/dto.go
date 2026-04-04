package whatsappbot

import "time"

// InboundMessage represents a standardized message received from a external provider.
type InboundMessage struct {
	From        string
	Body        string
	ProviderMsgID string
	ReceivedAt  time.Time
}

// OutboundReply represents a message we want to send back in the current context.
type OutboundReply struct {
	Body string
}
