package notification

import (
	"context"

	"github.com/google/uuid"
)

// PreferenceService resolves which channels are active for a given event and patient.
type PreferenceService struct {
	repo NotificationRepository
}

func NewPreferenceService(repo NotificationRepository) *PreferenceService {
	return &PreferenceService{repo: repo}
}

// ResolveActiveChannels returns channels the dispatcher should use for this payload.
// It checks patient preferences and verifies the patient has a valid address for each channel.
func (s *PreferenceService) ResolveActiveChannels(ctx context.Context, p AppointmentEventPayload) ([]string, error) {
	prefs, err := s.repo.GetPreferences(ctx, p.TenantID, p.PatientID)
	if err != nil {
		return nil, err
	}

	if !s.isEventEnabled(prefs, p.Event) {
		return nil, nil
	}

	var channels []string
	if prefs.EmailEnabled && p.PatientEmail != "" {
		channels = append(channels, ChannelEmail)
	}
	if prefs.WhatsAppEnabled && p.PatientPhone != "" {
		channels = append(channels, ChannelWhatsApp)
	}
	return channels, nil
}

// isEventEnabled checks whether the patient has opted in to this event type.
func (s *PreferenceService) isEventEnabled(prefs *PatientNotificationPreferences, event string) bool {
	switch event {
	case EventAppointmentCreated:
		return prefs.AppointmentCreatedEnabled
	case EventAppointmentConfirmed:
		return prefs.AppointmentConfirmedEnabled
	case EventAppointmentCanceled:
		return prefs.AppointmentCanceledEnabled
	case EventAppointmentRescheduled:
		return prefs.AppointmentRescheduledEnabled
	case EventAppointmentReminder:
		return prefs.ReminderEnabled
	}
	return true
}

// GetPreferences returns the stored (or default) preferences for a patient.
func (s *PreferenceService) GetPreferences(ctx context.Context, tenantID, patientID uuid.UUID) (*PatientNotificationPreferences, error) {
	return s.repo.GetPreferences(ctx, tenantID, patientID)
}

// UpsertPreferences saves or updates a patient's notification preferences.
func (s *PreferenceService) UpsertPreferences(ctx context.Context, tenantID, patientID uuid.UUID, req UpsertPreferencesRequest) (*PatientNotificationPreferences, error) {
	p := &PatientNotificationPreferences{
		ID:                           uuid.New(),
		TenantID:                     tenantID,
		PatientID:                    patientID,
		EmailEnabled:                 req.EmailEnabled,
		WhatsAppEnabled:              req.WhatsAppEnabled,
		ReminderEnabled:              req.ReminderEnabled,
		AppointmentCreatedEnabled:    req.AppointmentCreatedEnabled,
		AppointmentConfirmedEnabled:  req.AppointmentConfirmedEnabled,
		AppointmentCanceledEnabled:   req.AppointmentCanceledEnabled,
		AppointmentRescheduledEnabled: req.AppointmentRescheduledEnabled,
	}
	if err := s.repo.UpsertPreferences(ctx, p); err != nil {
		return nil, err
	}
	return s.repo.GetPreferences(ctx, tenantID, patientID)
}
