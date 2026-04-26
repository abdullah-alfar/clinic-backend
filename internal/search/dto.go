package search

import (
	"time"

	"github.com/google/uuid"
)

// SearchRequest holds all parameters needed to execute a global search.
// It is constructed by the handler and passed down to the service and every provider.
type SearchRequest struct {
	// TenantID scopes every query to a single tenant. Mandatory.
	TenantID uuid.UUID

	// Query is the trimmed search string.
	Query string

	// Types, when non-empty, restricts which providers are invoked.
	// Values must match EntityType constants (e.g. "patients", "doctors").
	Types []string

	// Limit is the maximum number of results per provider (default: DefaultLimitPerProvider).
	Limit int

	// --- Optional narrow filters ---

	// DateFrom restricts results to records on or after this date (UTC).
	DateFrom *time.Time

	// DateTo restricts results to records on or before this date (UTC).
	DateTo *time.Time

	// Status narrows results to a specific status string (e.g. "confirmed", "paid").
	Status string

	// PatientID narrows results to a specific patient (supported by providers that carry a patient FK).
	PatientID *uuid.UUID

	// DoctorID narrows results to a specific doctor.
	DoctorID *uuid.UUID
}

// SearchData is the top-level payload returned to the client.
type SearchData struct {
	Query    string              `json:"query"`
	Groups   []SearchResultGroup `json:"groups"`
	Warnings []string            `json:"warnings"`
}
