package mail

import (
	"context"
	"log"
)

// LogEmailSender is the dev/local implementation of EmailSender.
// It prints the email to stdout and always returns nil.
// Replace with SMTPSender, SendGridSender, or ResendSender in production.
type LogEmailSender struct{}

func NewLogEmailSender() *LogEmailSender {
	return &LogEmailSender{}
}

func (s *LogEmailSender) Send(_ context.Context, msg EmailMessage) error {
	log.Printf("[MAIL] 📧 To: %s | Subject: %s\n%s", msg.To, msg.Subject, msg.TextBody)
	return nil
}
