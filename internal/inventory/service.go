package inventory

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	CreateItem(ctx context.Context, tenantID uuid.UUID, req CreateItemReq) (*InventoryItem, error)
	GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*InventoryItem, error)
	ListItems(ctx context.Context, tenantID uuid.UUID) ([]InventoryItem, error)
	UpdateItem(ctx context.Context, tenantID, itemID uuid.UUID, req UpdateItemReq) (*InventoryItem, error)
	
	AdjustStock(ctx context.Context, tenantID, itemID uuid.UUID, userID *uuid.UUID, req AdjustStockReq) error
	ListMovements(ctx context.Context, tenantID, itemID uuid.UUID) ([]StockMovement, error)
}

type inventoryService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &inventoryService{repo: repo}
}

func (s *inventoryService) CreateItem(ctx context.Context, tenantID uuid.UUID, req CreateItemReq) (*InventoryItem, error) {
	item := &InventoryItem{
		TenantID:         tenantID,
		Name:             req.Name,
		SKU:              req.SKU,
		Unit:             req.Unit,
		CurrentStock:     req.CurrentStock,
		ReorderThreshold: req.ReorderThreshold,
		IsActive:         true,
	}

	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}

	// We should technically create a movement log for the initial stock if > 0
	if item.CurrentStock > 0 {
		reason := "Initial stock setup"
		err := s.repo.AdjustStockTx(ctx, nil, tenantID, item.ID, "in", item.CurrentStock, &reason, nil, nil, nil)
		if err != nil {
			// Not critical enough to fail item creation, but log ideally
		}
	}

	return item, nil
}

func (s *inventoryService) GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*InventoryItem, error) {
	return s.repo.GetItemByID(ctx, tenantID, itemID)
}

func (s *inventoryService) ListItems(ctx context.Context, tenantID uuid.UUID) ([]InventoryItem, error) {
	return s.repo.ListItems(ctx, tenantID)
}

func (s *inventoryService) UpdateItem(ctx context.Context, tenantID, itemID uuid.UUID, req UpdateItemReq) (*InventoryItem, error) {
	item, err := s.repo.GetItemByID(ctx, tenantID, itemID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.SKU != nil {
		item.SKU = req.SKU
	}
	if req.Unit != nil {
		item.Unit = *req.Unit
	}
	if req.ReorderThreshold != nil {
		item.ReorderThreshold = req.ReorderThreshold
	}
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *inventoryService) AdjustStock(ctx context.Context, tenantID, itemID uuid.UUID, userID *uuid.UUID, req AdjustStockReq) error {
	return s.repo.AdjustStockTx(ctx, nil, tenantID, itemID, req.MovementType, req.Quantity, req.Reason, req.VisitID, req.MedicalRecordID, userID)
}

func (s *inventoryService) ListMovements(ctx context.Context, tenantID, itemID uuid.UUID) ([]StockMovement, error) {
	return s.repo.ListMovements(ctx, tenantID, itemID)
}
