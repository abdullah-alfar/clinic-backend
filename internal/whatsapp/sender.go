package whatsapp

import "context"

// WhatsAppMessage is the canonical outbound message structure.
type WhatsAppMessage struct {
	To             string // E.164 phone number, e.g. +966501234567
	Body           string
	TemplateName   string   // for template-driven providers (Meta, Twilio)
	TemplateParams []string // positional template parameters
}

// WhatsAppSender abstracts all WhatsApp delivery providers.
// Implementations: LogWhatsAppSender (dev), MetaWhatsAppSender, TwilioWhatsAppSender (future).
type WhatsAppSender interface {
	Send(ctx context.Context, msg WhatsAppMessage) (providerMessageID string, err error)
}
