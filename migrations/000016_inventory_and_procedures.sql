CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) NULL,
    unit VARCHAR(50) NOT NULL,
    current_stock NUMERIC(12,2) NOT NULL DEFAULT 0,
    reorder_threshold NUMERIC(12,2) NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_items_tenant_id ON inventory_items(tenant_id);

CREATE TABLE IF NOT EXISTS inventory_stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    inventory_item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    visit_id UUID NULL,
    medical_record_id UUID NULL REFERENCES medical_records(id) ON DELETE SET NULL,
    movement_type VARCHAR(30) NOT NULL, -- in, out, adjustment
    quantity NUMERIC(12,2) NOT NULL,
    reason VARCHAR(255) NULL,
    created_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_stock_movements_tenant_item ON inventory_stock_movements(tenant_id, inventory_item_id);
CREATE INDEX idx_inventory_stock_movements_medical_record ON inventory_stock_movements(medical_record_id);

CREATE TABLE IF NOT EXISTS procedure_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_procedure_catalog_tenant_id ON procedure_catalog(tenant_id);

CREATE TABLE IF NOT EXISTS procedure_catalog_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    procedure_catalog_id UUID NOT NULL REFERENCES procedure_catalog(id) ON DELETE CASCADE,
    inventory_item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE RESTRICT,
    quantity NUMERIC(12,2) NOT NULL,
    UNIQUE(procedure_catalog_id, inventory_item_id)
);

CREATE TABLE IF NOT EXISTS medical_record_procedures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    medical_record_id UUID NOT NULL REFERENCES medical_records(id) ON DELETE CASCADE,
    procedure_catalog_id UUID NOT NULL REFERENCES procedure_catalog(id) ON DELETE RESTRICT,
    performed_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_medical_record_procedures_record ON medical_record_procedures(medical_record_id);
CREATE INDEX idx_medical_record_procedures_tenant ON medical_record_procedures(tenant_id);
