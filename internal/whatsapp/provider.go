package whatsapp

import (
	"context"
	"fmt"
	"log"
)

// LogWhatsAppSender is the dev/local implementation.
// Logs messages to stdout. Never actually sends anything.
type LogWhatsAppSender struct{}

func NewLogWhatsAppSender() *LogWhatsAppSender { return &LogWhatsAppSender{} }

func (s *LogWhatsAppSender) Send(_ context.Context, msg WhatsAppMessage) (string, error) {
	log.Printf("[WHATSAPP] 💬 To: %s\n%s", msg.To, msg.Body)
	return fmt.Sprintf("fake-msg-id-%s", msg.To), nil
}

// MetaWhatsAppSender is a stub for the Meta Cloud API provider.
// Implement when moving to production.
// See: https://developers.facebook.com/docs/whatsapp/cloud-api
type MetaWhatsAppSender struct {
	PhoneNumberID string
	AccessToken   string
}

func NewMetaWhatsAppSender(phoneNumberID, accessToken string) *MetaWhatsAppSender {
	return &MetaWhatsAppSender{PhoneNumberID: phoneNumberID, AccessToken: accessToken}
}

func (s *MetaWhatsAppSender) Send(_ context.Context, msg WhatsAppMessage) (string, error) {
	// TODO: POST https://graph.facebook.com/v18.0/{PhoneNumberID}/messages
	// with Authorization: Bearer {AccessToken}
	log.Printf("[WHATSAPP META STUB] Would send to %s: %s", msg.To, msg.Body)
	return "", nil
}
