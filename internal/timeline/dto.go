package timeline

import (
	"time"

	"github.com/google/uuid"
)

type TimelineItemDTO struct {
	ID          uuid.UUID      `json:"id"`
	Type        TimelineItemType `json:"type"`
	Title       string         `json:"title"`
	Subtitle    string         `json:"subtitle"`
	Description string         `json:"description"`
	OccurredAt  time.Time      `json:"occurred_at"`
	Status      *string        `json:"status,omitempty"`
	EntityID    uuid.UUID      `json:"entity_id"`
	EntityURL   string         `json:"entity_url"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type TimelineResponseDTO struct {
	PatientID uuid.UUID         `json:"patient_id"`
	Items     []TimelineItemDTO `json:"items"`
}
