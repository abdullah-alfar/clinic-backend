package notification

import "fmt"

// TemplateData is the normalised context passed to all message builders.
type TemplateData struct {
	PatientName     string
	DoctorName      string
	ClinicName      string
	AppointmentDate string
	AppointmentTime string
	Timezone        string
}

// EmailTemplate holds the rendered subject and body for an outbound email.
type EmailTemplate struct {
	Subject  string
	TextBody string
	HTMLBody string
}

// BuildEmailTemplate returns a rendered email for the given event and data.
func BuildEmailTemplate(event string, d TemplateData) EmailTemplate {
	switch event {
	case EventAppointmentCreated:
		subj := fmt.Sprintf("Appointment Booked — %s", d.ClinicName)
		body := fmt.Sprintf(
			"Dear %s,\n\nYour appointment has been booked.\n\nDoctor: %s\nDate: %s\nTime: %s (%s)\n\nThank you,\n%s",
			d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone, d.ClinicName)
		return EmailTemplate{Subject: subj, TextBody: body, HTMLBody: simpleHTML(subj, body)}

	case EventAppointmentConfirmed:
		subj := fmt.Sprintf("Appointment Confirmed — %s", d.ClinicName)
		body := fmt.Sprintf(
			"Dear %s,\n\nYour appointment is confirmed.\n\nDoctor: %s\nDate: %s\nTime: %s (%s)\n\nSee you soon,\n%s",
			d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone, d.ClinicName)
		return EmailTemplate{Subject: subj, TextBody: body, HTMLBody: simpleHTML(subj, body)}

	case EventAppointmentCanceled:
		subj := fmt.Sprintf("Appointment Canceled — %s", d.ClinicName)
		body := fmt.Sprintf(
			"Dear %s,\n\nYour appointment on %s at %s with Dr. %s has been canceled.\n\nPlease contact us to rebook.\n\n%s",
			d.PatientName, d.AppointmentDate, d.AppointmentTime, d.DoctorName, d.ClinicName)
		return EmailTemplate{Subject: subj, TextBody: body, HTMLBody: simpleHTML(subj, body)}

	case EventAppointmentRescheduled:
		subj := fmt.Sprintf("Appointment Rescheduled — %s", d.ClinicName)
		body := fmt.Sprintf(
			"Dear %s,\n\nYour appointment has been rescheduled.\n\nDoctor: %s\nNew Date: %s\nNew Time: %s (%s)\n\nThank you,\n%s",
			d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone, d.ClinicName)
		return EmailTemplate{Subject: subj, TextBody: body, HTMLBody: simpleHTML(subj, body)}

	case EventAppointmentReminder:
		subj := fmt.Sprintf("Appointment Reminder — %s", d.ClinicName)
		body := fmt.Sprintf(
			"Dear %s,\n\nReminder: You have an appointment tomorrow.\n\nDoctor: %s\nDate: %s\nTime: %s (%s)\n\nSee you soon,\n%s",
			d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone, d.ClinicName)
		return EmailTemplate{Subject: subj, TextBody: body, HTMLBody: simpleHTML(subj, body)}
	}

	return EmailTemplate{
		Subject:  "Clinic Notification — " + d.ClinicName,
		TextBody: "You have a new notification from " + d.ClinicName,
		HTMLBody:  "<p>You have a new notification from " + d.ClinicName + "</p>",
	}
}

// BuildWhatsAppMessage returns a plain-text WhatsApp body for the given event.
func BuildWhatsAppMessage(event string, d TemplateData) string {
	switch event {
	case EventAppointmentCreated:
		return fmt.Sprintf("*%s*\n\nHello %s 👋\n\nYour appointment has been booked!\n\n👨‍⚕️ Doctor: %s\n📅 Date: %s\n🕐 Time: %s (%s)",
			d.ClinicName, d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone)
	case EventAppointmentConfirmed:
		return fmt.Sprintf("*%s*\n\nHello %s ✅\n\nYour appointment is confirmed!\n\n👨‍⚕️ Doctor: %s\n📅 Date: %s\n🕐 Time: %s (%s)",
			d.ClinicName, d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone)
	case EventAppointmentCanceled:
		return fmt.Sprintf("*%s*\n\nHello %s,\n\nYour appointment on %s at %s has been canceled.\n\nPlease contact us to rebook. 📞",
			d.ClinicName, d.PatientName, d.AppointmentDate, d.AppointmentTime)
	case EventAppointmentRescheduled:
		return fmt.Sprintf("*%s*\n\nHello %s 🔄\n\nYour appointment has been rescheduled.\n\n👨‍⚕️ Doctor: %s\n📅 New Date: %s\n🕐 New Time: %s (%s)",
			d.ClinicName, d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone)
	case EventAppointmentReminder:
		return fmt.Sprintf("*%s*\n\nHello %s ⏰\n\nReminder: You have an appointment TOMORROW!\n\n👨‍⚕️ Doctor: %s\n📅 Date: %s\n🕐 Time: %s (%s)",
			d.ClinicName, d.PatientName, d.DoctorName, d.AppointmentDate, d.AppointmentTime, d.Timezone)
	}
	return fmt.Sprintf("*%s*\n\nYou have a new notification.", d.ClinicName)
}

// simpleHTML wraps plain text in minimal HTML for email clients.
func simpleHTML(heading, body string) string {
	return fmt.Sprintf(
		`<!DOCTYPE html><html><body style="font-family:sans-serif;line-height:1.6;max-width:600px;margin:0 auto;padding:20px"><h2 style="color:#1a1a2e">%s</h2><pre style="white-space:pre-wrap;font-family:inherit">%s</pre></body></html>`,
		heading, body)
}
