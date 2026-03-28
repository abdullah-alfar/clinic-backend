-- Safe alteration of attachments table
-- Add new columns with defaults to prevent breaking existing rows, then alter to remove default if appropriate (or keep standard defaults).
ALTER TABLE attachments 
    ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT 'Uploaded File',
    ADD COLUMN IF NOT EXISTS mime_type VARCHAR(100) NOT NULL DEFAULT 'application/octet-stream',
    ADD COLUMN IF NOT EXISTS file_size BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Create Indexes for performance on attachments
CREATE INDEX IF NOT EXISTS idx_attachments_tenant_patient ON attachments(tenant_id, patient_id);
CREATE INDEX IF NOT EXISTS idx_attachments_tenant_appointment ON attachments(tenant_id, appointment_id);

-- Report AI Analyses Table
CREATE TABLE IF NOT EXISTS report_ai_analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
    analysis_type VARCHAR(50) NOT NULL, -- 'summary', 'extraction', 'qna'
    status VARCHAR(30) NOT NULL, -- 'pending', 'completed', 'failed'
    summary TEXT,
    structured_data JSONB,
    raw_response JSONB,
    error_message TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for AI analyses
CREATE INDEX IF NOT EXISTS idx_repor_ai_tenant_patient ON report_ai_analyses(tenant_id, patient_id);
CREATE INDEX IF NOT EXISTS idx_repor_ai_tenant_attachment ON report_ai_analyses(tenant_id, attachment_id);
