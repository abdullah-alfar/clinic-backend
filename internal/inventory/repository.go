package inventory

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	CreateItem(ctx context.Context, item *InventoryItem) error
	GetItemByID(ctx context.Context, tenantID, itemID uuid.UUID) (*InventoryItem, error)
	ListItems(ctx context.Context, tenantID uuid.UUID) ([]InventoryItem, error)
	UpdateItem(ctx context.Context, item *InventoryItem) error
	
	AdjustStockTx(ctx context.Context, tx *sql.Tx, tenantID, itemID uuid.UUID, movementType string, quantity float64, reason *string, visitID, recordID, userID *uuid.UUID) error
	ListMovements(ctx context.Context, tenantID, itemID uuid.UUID) ([]StockMovement, error)
	
	BeginTx(ctx context.Context) (*sql.Tx, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *PostgresRepository) CreateItem(ctx context.Context, item *InventoryItem) error {
	query := `
		INSERT INTO inventory_items (tenant_id, name, sku, unit, current_stock, reorder_threshold, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		item.TenantID, item.Name, item.SKU, item.Unit, item.CurrentStock, item.ReorderThreshold, item.IsActive,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (r *PostgresRepository) GetItemByID(ctx context.Context, tenantID, itemID uuid.UUID) (*InventoryItem, error) {
	query := `
		SELECT id, tenant_id, name, sku, unit, current_stock, reorder_threshold, is_active, created_at, updated_at
		FROM inventory_items
		WHERE id = $1 AND tenant_id = $2
	`
	item := &InventoryItem{}
	err := r.db.QueryRowContext(ctx, query, itemID, tenantID).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.SKU, &item.Unit,
		&item.CurrentStock, &item.ReorderThreshold, &item.IsActive,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *PostgresRepository) ListItems(ctx context.Context, tenantID uuid.UUID) ([]InventoryItem, error) {
	query := `
		SELECT id, tenant_id, name, sku, unit, current_stock, reorder_threshold, is_active, created_at, updated_at
		FROM inventory_items
		WHERE tenant_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []InventoryItem
	for rows.Next() {
		var item InventoryItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.SKU, &item.Unit,
			&item.CurrentStock, &item.ReorderThreshold, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *PostgresRepository) UpdateItem(ctx context.Context, item *InventoryItem) error {
	query := `
		UPDATE inventory_items
		SET name = $1, sku = $2, unit = $3, reorder_threshold = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7
		RETURNING updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		item.Name, item.SKU, item.Unit, item.ReorderThreshold, item.IsActive, item.ID, item.TenantID,
	).Scan(&item.UpdatedAt)
}

func (r *PostgresRepository) AdjustStockTx(ctx context.Context, tx *sql.Tx, tenantID, itemID uuid.UUID, movementType string, quantity float64, reason *string, visitID, recordID, userID *uuid.UUID) error {
	
	// Ensure positive quantity for DB
	if quantity < 0 {
		return errors.New("quantity must be positive")
	}

	// Calculate stock change
	var diff float64
	if movementType == "in" {
		diff = quantity
	} else if movementType == "out" {
		diff = -quantity
	} else {
		// "adjustment" can be handled separately, assuming positive sets exact stock, but usually adjustments are +/-
		// For simplicity, we treat "adjustment" as a raw diff input but standard movement types are safer.
		// Let's assume adjustment can be positive or negative depending on input?
		// Better: standard is 'in' or 'out'
		diff = quantity 
	}

	// Update stock
	updateQuery := `
		UPDATE inventory_items
		SET current_stock = current_stock + $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, updateQuery, diff, itemID, tenantID)
	} else {
		_, err = r.db.ExecContext(ctx, updateQuery, diff, itemID, tenantID)
	}
	if err != nil {
		return err
	}

	// Record movement
	insertQuery := `
		INSERT INTO inventory_stock_movements (
			tenant_id, inventory_item_id, visit_id, medical_record_id, movement_type, quantity, reason, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if tx != nil {
		_, err = tx.ExecContext(ctx, insertQuery, tenantID, itemID, visitID, recordID, movementType, quantity, reason, userID)
	} else {
		_, err = r.db.ExecContext(ctx, insertQuery, tenantID, itemID, visitID, recordID, movementType, quantity, reason, userID)
	}
	return err
}

func (r *PostgresRepository) ListMovements(ctx context.Context, tenantID, itemID uuid.UUID) ([]StockMovement, error) {
	query := `
		SELECT id, tenant_id, inventory_item_id, visit_id, medical_record_id, movement_type, quantity, reason, created_by, created_at
		FROM inventory_stock_movements
		WHERE tenant_id = $1 AND inventory_item_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movements []StockMovement
	for rows.Next() {
		var sm StockMovement
		if err := rows.Scan(
			&sm.ID, &sm.TenantID, &sm.InventoryItemID, &sm.VisitID, &sm.MedicalRecordID,
			&sm.MovementType, &sm.Quantity, &sm.Reason, &sm.CreatedBy, &sm.CreatedAt,
		); err != nil {
			return nil, err
		}
		movements = append(movements, sm)
	}
	return movements, nil
}
