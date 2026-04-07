package inventory

import "github.com/google/uuid"

type CreateItemReq struct {
	Name             string   `json:"name"`
	SKU              *string  `json:"sku"`
	Unit             string   `json:"unit"`
	CurrentStock     float64  `json:"current_stock"`
	ReorderThreshold *float64 `json:"reorder_threshold"`
}

type UpdateItemReq struct {
	Name             *string  `json:"name"`
	SKU              *string  `json:"sku"`
	Unit             *string  `json:"unit"`
	ReorderThreshold *float64 `json:"reorder_threshold"`
	IsActive         *bool    `json:"is_active"`
}

type AdjustStockReq struct {
	MovementType    string     `json:"movement_type"` // "in", "out", "adjustment"
	Quantity        float64    `json:"quantity"`
	Reason          *string    `json:"reason"`
	VisitID         *uuid.UUID `json:"visit_id"`
	MedicalRecordID *uuid.UUID `json:"medical_record_id"`
}
