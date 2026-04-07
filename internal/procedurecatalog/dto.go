package procedurecatalog

import "github.com/google/uuid"

type ProcedureItemReq struct {
	InventoryItemID uuid.UUID `json:"inventory_item_id"`
	Quantity        float64   `json:"quantity"`
}

type CreateProcedureReq struct {
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Items       []ProcedureItemReq `json:"items"`
}

type UpdateProcedureReq struct {
	Name        *string            `json:"name"`
	Description *string            `json:"description"`
	IsActive    *bool              `json:"is_active"`
	Items       []ProcedureItemReq `json:"items"` // If provided, completely overwrites old items
}
