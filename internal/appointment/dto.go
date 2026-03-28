package appointment

import (
	"time"

	"github.com/google/uuid"
)

// CalendarQueryParams holds validated parameters for the calendar query.
type CalendarQueryParams struct {
	DateFrom time.Time
	DateTo   time.Time
	DoctorID *uuid.UUID
}

// CalendarAppointmentDTO is the enriched response shape sent to the frontend calendar.
// It includes joined patient and doctor names to avoid N+1 lookups on the client.
type CalendarAppointmentDTO struct {
	ID          string  `json:"id"`
	PatientID   string  `json:"patient_id"`
	PatientName string  `json:"patient_name"`
	DoctorID    string  `json:"doctor_id"`
	DoctorName  string  `json:"doctor_name"`
	Status      string  `json:"status"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Reason      *string `json:"reason"`
}

// RescheduleRequest is the typed body for the dedicated reschedule endpoint.
type RescheduleRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// CalendarResponse is the top-level envelope returned by HandleGetCalendar.
type CalendarResponse struct {
	Data     []CalendarAppointmentDTO `json:"data"`
	Timezone string                   `json:"timezone"`
	Message  string                   `json:"message"`
	Error    *string                  `json:"error"`
}

// toCalendarDTO maps a CalendarAppointment domain model to its DTO representation.
func toCalendarDTO(a CalendarAppointment) CalendarAppointmentDTO {
	return CalendarAppointmentDTO{
		ID:          a.ID.String(),
		PatientID:   a.PatientID.String(),
		PatientName: a.PatientName,
		DoctorID:    a.DoctorID.String(),
		DoctorName:  a.DoctorName,
		Status:      a.Status,
		StartTime:   a.StartTime.UTC().Format(time.RFC3339),
		EndTime:     a.EndTime.UTC().Format(time.RFC3339),
		Reason:      a.Reason,
	}
}
