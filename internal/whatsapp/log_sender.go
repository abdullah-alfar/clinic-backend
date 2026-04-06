package whatsapp

import (
	"context"
	"fmt"
	"log"
)

// LogWhatsAppSender is the dev/local implementation.
// Logs messages to stdout. Never actually sends anything.
type LogWhatsAppSender struct{}

func NewLogWhatsAppSender() *LogWhatsAppSender {
	return &LogWhatsAppSender{}
}

func (s *LogWhatsAppSender) Send(_ context.Context, msg WhatsAppMessage) (string, error) {
	log.Printf("[WHATSAPP] 💬 To: %s\n%s", msg.To, msg.Body)
	return fmt.Sprintf("fake-msg-id-%s", msg.To), nil
}
