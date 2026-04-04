package mail

import "context"

// EmailMessage is the canonical structure for all outbound emails.
// All template builders and senders operate on this type.
type EmailMessage struct {
	To       string
	From     string
	Subject  string
	TextBody string
	HTMLBody string
}

// EmailSender is the delivery abstraction for all email providers.
// Implementations: LogEmailSender (dev), SMTPSender, SendGridSender, ResendSender (future).
// The service layer depends only on this interface, never on concrete providers.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}
