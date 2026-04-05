package search

import (
	"context"

	"github.com/google/uuid"
)

type EntityType string

const (
	EntityPatient      EntityType = "patients"
	EntityDoctor       EntityType = "doctors"
	EntityAppointment  EntityType = "appointments"
	EntityInvoice      EntityType = "invoices"
	EntityReport       EntityType = "reports"
	EntityNote         EntityType = "notes"
	EntityNotification EntityType = "notifications"
	EntityMemory       EntityType = "memory"
	EntityAudit        EntityType = "audit_logs"
	EntityDoctorSchedule EntityType = "doctor_schedules"
)

type SearchResultItem struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Subtitle    string         `json:"subtitle"`
	Description string         `json:"description"`
	URL         string         `json:"url"`
	Score       float64        `json:"score"`
	Metadata    map[string]any `json:"metadata"`
}

type SearchResultGroup struct {
	Type    string             `json:"type"`
	Label   string             `json:"label"`
	Count   int                `json:"count"`
	Results []SearchResultItem `json:"results"`
}

type SearchProvider interface {
	GetEntityType() EntityType
	GetEntityLabel() string
	Search(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]SearchResultItem, error)
}
