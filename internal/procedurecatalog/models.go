package procedurecatalog

import (
	"time"

	"github.com/google/uuid"
)

type ProcedureCatalog struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	IsActive    bool                   `json:"is_active"`
	Items       []ProcedureCatalogItem `json:"items"` // Joined
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ProcedureCatalogItem struct {
	ID                 uuid.UUID `json:"id"`
	ProcedureCatalogID uuid.UUID `json:"procedure_catalog_id"`
	InventoryItemID    uuid.UUID `json:"inventory_item_id"`
	Quantity           float64   `json:"quantity"`
	// Additional payload for easy rendering on frontend
	InventoryItemName *string  `json:"inventory_item_name,omitempty"`
	Unit              *string  `json:"unit,omitempty"`
	CurrentStock      *float64 `json:"current_stock,omitempty"`
}
