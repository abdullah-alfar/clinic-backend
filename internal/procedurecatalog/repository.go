package procedurecatalog

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Repository interface {
	CreateProcedure(ctx context.Context, proc *ProcedureCatalog) error
	GetProcedureByID(ctx context.Context, tenantID, procID uuid.UUID) (*ProcedureCatalog, error)
	ListProcedures(ctx context.Context, tenantID uuid.UUID) ([]ProcedureCatalog, error)
	UpdateProcedure(ctx context.Context, proc *ProcedureCatalog) error
	SetProcedureItems(ctx context.Context, procID uuid.UUID, items []ProcedureItemReq) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateProcedure(ctx context.Context, proc *ProcedureCatalog) error {
	query := `
		INSERT INTO procedure_catalog (tenant_id, name, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		proc.TenantID, proc.Name, proc.Description, proc.IsActive,
	).Scan(&proc.ID, &proc.CreatedAt, &proc.UpdatedAt)
}

func (r *PostgresRepository) GetProcedureByID(ctx context.Context, tenantID, procID uuid.UUID) (*ProcedureCatalog, error) {
	query := `
		SELECT id, tenant_id, name, description, is_active, created_at, updated_at
		FROM procedure_catalog
		WHERE id = $1 AND tenant_id = $2
	`
	proc := &ProcedureCatalog{}
	err := r.db.QueryRowContext(ctx, query, procID, tenantID).Scan(
		&proc.ID, &proc.TenantID, &proc.Name, &proc.Description, &proc.IsActive,
		&proc.CreatedAt, &proc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch items
	itemsQuery := `
		SELECT pci.id, pci.procedure_catalog_id, pci.inventory_item_id, pci.quantity,
		       ii.name, ii.unit, ii.current_stock
		FROM procedure_catalog_items pci
		JOIN inventory_items ii ON pci.inventory_item_id = ii.id
		WHERE pci.procedure_catalog_id = $1
	`
	rows, err := r.db.QueryContext(ctx, itemsQuery, proc.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item ProcedureCatalogItem
		if err := rows.Scan(
			&item.ID, &item.ProcedureCatalogID, &item.InventoryItemID, &item.Quantity,
			&item.InventoryItemName, &item.Unit, &item.CurrentStock,
		); err != nil {
			return nil, err
		}
		proc.Items = append(proc.Items, item)
	}

	return proc, nil
}

func (r *PostgresRepository) ListProcedures(ctx context.Context, tenantID uuid.UUID) ([]ProcedureCatalog, error) {
	query := `
		SELECT id, tenant_id, name, description, is_active, created_at, updated_at
		FROM procedure_catalog
		WHERE tenant_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var procs []ProcedureCatalog
	for rows.Next() {
		var proc ProcedureCatalog
		if err := rows.Scan(
			&proc.ID, &proc.TenantID, &proc.Name, &proc.Description, &proc.IsActive,
			&proc.CreatedAt, &proc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		procs = append(procs, proc)
	}

	// For efficiency in a list view, we might optionally load items. Let's do it individually or joined.
	// We'll skip item loading in list view unless required, or we could do an IN query.
	// For simplicity, we just return the templates here.

	return procs, nil
}

func (r *PostgresRepository) UpdateProcedure(ctx context.Context, proc *ProcedureCatalog) error {
	query := `
		UPDATE procedure_catalog
		SET name = $1, description = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5
		RETURNING updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		proc.Name, proc.Description, proc.IsActive, proc.ID, proc.TenantID,
	).Scan(&proc.UpdatedAt)
}

func (r *PostgresRepository) SetProcedureItems(ctx context.Context, procID uuid.UUID, items []ProcedureItemReq) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing
	delQ := `DELETE FROM procedure_catalog_items WHERE procedure_catalog_id = $1`
	if _, err := tx.ExecContext(ctx, delQ, procID); err != nil {
		return err
	}

	// Insert new
	insQ := `
		INSERT INTO procedure_catalog_items (procedure_catalog_id, inventory_item_id, quantity)
		VALUES ($1, $2, $3)
	`
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, insQ, procID, item.InventoryItemID, item.Quantity); err != nil {
			return err
		}
	}

	return tx.Commit()
}
