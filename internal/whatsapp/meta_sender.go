package whatsapp

import (
	"context"
	"log"
)

// MetaWhatsAppSender is a stub for the Meta Cloud API provider.
type MetaWhatsAppSender struct {
	PhoneNumberID string
	AccessToken   string
}

func NewMetaWhatsAppSender(phoneNumberID, accessToken string) *MetaWhatsAppSender {
	return &MetaWhatsAppSender{
		PhoneNumberID: phoneNumberID,
		AccessToken:   accessToken,
	}
}

func (s *MetaWhatsAppSender) Send(_ context.Context, msg WhatsAppMessage) (string, error) {
	log.Printf("[WHATSAPP META STUB] Would send to %s: %s", msg.To, msg.Body)
	return "", nil
}
