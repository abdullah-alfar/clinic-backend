package timeline

import (
	"time"

	"github.com/google/uuid"
)

type TimelineItemType string

const (
	TypeAppointment    TimelineItemType = "appointment"
	TypeMedicalRecord  TimelineItemType = "medical_record"
	TypeInvoice        TimelineItemType = "invoice"
	TypePayment        TimelineItemType = "payment"
	TypeNote           TimelineItemType = "note"
	TypeNotification   TimelineItemType = "notification"
	TypeAttachment     TimelineItemType = "attachment"
	TypeDocument       TimelineItemType = "document"
)

type TimelineItem struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	PatientID   uuid.UUID
	Type        TimelineItemType
	Title       string
	Subtitle    string
	Description string
	OccurredAt  time.Time
	Status      *string
	EntityID    uuid.UUID
	EntityURL   string
	Metadata    map[string]any
}
