package mail

import (
	"log"
)

type Mailer interface {
	SendReminder(toEmail, patientName, appointmentTime string) error
	SendConfirmation(toEmail, patientName, appointmentTime string) error
}

// LocalConsoleMailer implements Mailer for dev environments
type LocalConsoleMailer struct{}

func NewLocalConsoleMailer() *LocalConsoleMailer {
	return &LocalConsoleMailer{}
}

func (m *LocalConsoleMailer) SendReminder(toEmail, patientName, timeStr string) error {
	log.Printf("[MAILER] 📧 Sending REMINDER to %s: Hello %s, you have an appointment at %s.\n", toEmail, patientName, timeStr)
	return nil
}

func (m *LocalConsoleMailer) SendConfirmation(toEmail, patientName, timeStr string) error {
	log.Printf("[MAILER] 📧 Sending CONFIRMATION to %s: Hello %s, your appointment is confirmed for %s.\n", toEmail, patientName, timeStr)
	return nil
}
