package notification

import (
	"context"

	"clinic-backend/internal/queue"
	"github.com/google/uuid"
)

// Dispatcher is the public interface used by domain services (like appointment.Service).
type Dispatcher interface {
	Dispatch(ctx context.Context, p AppointmentEventPayload) error
}

// NotificationDispatcher handles routing an event payload to the correct channels
// based on patient preferences, creating database tracking records, and enqueuing jobs.
type NotificationDispatcher struct {
	repo  NotificationRepository
	prefs *PreferenceService
	queue *queue.QueueClient
}

func NewNotificationDispatcher(repo NotificationRepository, prefs *PreferenceService, q *queue.QueueClient) *NotificationDispatcher {
	return &NotificationDispatcher{
		repo:  repo,
		prefs: prefs,
		queue: q,
	}
}

// Dispatch processes an appointment event payload.
func (d *NotificationDispatcher) Dispatch(ctx context.Context, p AppointmentEventPayload) error {
	// First, let the preference service figure out where this needs to go.
	channels, err := d.prefs.ResolveActiveChannels(ctx, p)
	if err != nil {
		return err
	}

	if len(channels) == 0 {
		return nil // Client opted out or missing contact info.
	}

	// Prepare template data to render messages.
	tData := TemplateData{
		PatientName:     p.PatientName,
		DoctorName:      p.DoctorName,
		ClinicName:      p.ClinicName,
		AppointmentDate: p.StartTime.Format("Jan 02, 2006"),
		AppointmentTime: p.StartTime.Format("15:04"),
		Timezone:        p.Timezone,
	}

	// Route to each enabled channel.
	for _, c := range channels {
		switch c {
		case ChannelEmail:
			if err := d.dispatchEmail(ctx, p, tData); err != nil {
				return err
			}
		case ChannelWhatsApp:
			if err := d.dispatchWhatsApp(ctx, p, tData); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *NotificationDispatcher) dispatchEmail(ctx context.Context, p AppointmentEventPayload, tData TemplateData) error {
	tmpl := BuildEmailTemplate(p.Event, tData)

	n := &OutboundNotification{
		ID:            uuid.New(),
		TenantID:      p.TenantID,
		PatientID:     &p.PatientID,
		AppointmentID: &p.AppointmentID,
		Channel:       ChannelEmail,
		EventType:     p.Event,
		Recipient:     p.PatientEmail,
		Subject:       &tmpl.Subject,
		Message:       tmpl.HTMLBody,
		Status:        StatusPending,
	}

	if err := d.repo.CreateNotification(ctx, n); err != nil {
		return err
	}

	if d.queue != nil {
		return d.queue.EnqueueEmailNotification(queue.EmailNotificationPayload{
			NotificationID: n.ID.String(),
			TenantID:       n.TenantID.String(),
			To:             n.Recipient,
			Subject:        tmpl.Subject,
			HTMLBody:       tmpl.HTMLBody,
			TextBody:       tmpl.TextBody,
		})
	}
	return nil
}

func (d *NotificationDispatcher) dispatchWhatsApp(ctx context.Context, p AppointmentEventPayload, tData TemplateData) error {
	msg := BuildWhatsAppMessage(p.Event, tData)

	n := &OutboundNotification{
		ID:            uuid.New(),
		TenantID:      p.TenantID,
		PatientID:     &p.PatientID,
		AppointmentID: &p.AppointmentID,
		Channel:       ChannelWhatsApp,
		EventType:     p.Event,
		Recipient:     p.PatientPhone,
		Message:       msg,
		Status:        StatusPending,
	}

	if err := d.repo.CreateNotification(ctx, n); err != nil {
		return err
	}

	if d.queue != nil {
		return d.queue.EnqueueWhatsAppNotification(queue.WhatsAppNotificationPayload{
			NotificationID: n.ID.String(),
			TenantID:       n.TenantID.String(),
			To:             n.Recipient,
			Body:           n.Message,
		})
	}
	return nil
}
