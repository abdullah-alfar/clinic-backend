package inventory

import (
	"time"

	"github.com/google/uuid"
)

type InventoryItem struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Name             string     `json:"name"`
	SKU              *string    `json:"sku"`
	Unit             string     `json:"unit"`
	CurrentStock     float64    `json:"current_stock"`
	ReorderThreshold *float64   `json:"reorder_threshold"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type StockMovement struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	InventoryItemID uuid.UUID  `json:"inventory_item_id"`
	VisitID         *uuid.UUID `json:"visit_id"`
	MedicalRecordID *uuid.UUID `json:"medical_record_id"`
	MovementType    string     `json:"movement_type"` // "in", "out", "adjustment"
	Quantity        float64    `json:"quantity"`
	Reason          *string    `json:"reason"`
	CreatedBy       *uuid.UUID `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
}
