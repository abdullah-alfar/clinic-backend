-- Migration: Add documents table
-- Created at: 2026-04-09

CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    appointment_id UUID REFERENCES appointments(id) ON DELETE SET NULL,
    medical_record_id UUID REFERENCES medical_records(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('lab_report', 'prescription', 'id_document', 'insurance', 'consent_form', 'general')),
    uploaded_by UUID,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_documents_tenant_patient ON documents(tenant_id, patient_id);
CREATE INDEX idx_documents_category ON documents(category);
CREATE INDEX idx_documents_appointment ON documents(appointment_id);
CREATE INDEX idx_documents_medical_record ON documents(medical_record_id);

-- Add trigger for updated_at if it exists in the system (typical for this project)
-- If not, we'll handle it in Go. Assuming there's a common trigger.
-- Let's check 000001_init.sql for trigger patterns.
