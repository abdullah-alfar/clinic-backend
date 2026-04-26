package search

import "context"

// MinQueryLength is the minimum number of characters required to execute a search.
const MinQueryLength = 2

// DefaultLimitPerProvider is the default number of results returned per provider when no limit is specified.
const DefaultLimitPerProvider = 20

// MaxLimitPerProvider caps the per-provider result set to prevent heavy queries.
const MaxLimitPerProvider = 50

// EntityType identifies the domain category of a search result.
type EntityType string

const (
	EntityPatient        EntityType = "patients"
	EntityDoctor         EntityType = "doctors"
	EntityAppointment    EntityType = "appointments"
	EntityInvoice        EntityType = "invoices"
	EntityReport         EntityType = "reports"
	EntityNote           EntityType = "notes"
	EntityNotification   EntityType = "notifications"
	EntityMemory         EntityType = "memory"
	EntityAudit          EntityType = "audit_logs"
	EntityDoctorSchedule EntityType = "doctor_schedules"
)

// providerPriority defines the display order of result groups.
// Lower index = higher priority = appears first.
var providerPriority = []EntityType{
	EntityPatient,
	EntityDoctor,
	EntityAppointment,
	EntityInvoice,
	EntityNote,
	EntityReport,
	EntityMemory,
	EntityNotification,
	EntityDoctorSchedule,
	EntityAudit,
}

// priorityOf returns the ordinal position of an entity type.
// Unknown types are sorted to the end.
func priorityOf(et EntityType) int {
	for i, p := range providerPriority {
		if p == et {
			return i
		}
	}
	return len(providerPriority)
}

// SearchResultItem is a single normalised search hit returned by any provider.
type SearchResultItem struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Subtitle    string         `json:"subtitle"`
	Description string         `json:"description"`
	URL         string         `json:"url"`
	Score       float64        `json:"score"`
	Metadata    map[string]any `json:"metadata"`
}

// SearchResultGroup is a provider's result set, grouped by entity type.
type SearchResultGroup struct {
	Type    string             `json:"type"`
	Label   string             `json:"label"`
	Count   int                `json:"count"`
	Results []SearchResultItem `json:"results"`
}

// SearchProvider is the contract that every domain-specific search provider must satisfy.
// Providers are responsible for:
//   - Scoping queries to the tenant in SearchRequest.TenantID.
//   - Applying a safe result LIMIT via SearchRequest.Limit.
//   - Returning normalised SearchResultItems (Score may be left at 0; the Ranker will enrich it).
type SearchProvider interface {
	// Type returns the canonical EntityType for this provider.
	Type() EntityType

	// Label returns the human-readable group label shown in the UI.
	Label() string

	// Search executes a scoped, parameterised query against this provider's domain.
	// It must honour ctx cancellation (pass ctx to every DB call).
	Search(ctx context.Context, req SearchRequest) ([]SearchResultItem, error)
}
