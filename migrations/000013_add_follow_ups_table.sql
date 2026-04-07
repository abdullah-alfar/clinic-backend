CREATE TABLE IF NOT EXISTS follow_ups (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    doctor_id UUID,
    appointment_id UUID,
    medical_record_id UUID,
    reason TEXT NOT NULL,
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    priority VARCHAR(10) DEFAULT 'medium',
    auto_generated BOOLEAN DEFAULT FALSE,
    created_by UUID NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (doctor_id) REFERENCES doctors(id),
    FOREIGN KEY (appointment_id) REFERENCES appointments(id)
);

CREATE INDEX idx_followups_tenant ON follow_ups(tenant_id);
CREATE INDEX idx_followups_patient ON follow_ups(patient_id);
CREATE INDEX idx_followups_status ON follow_ups(status);
CREATE INDEX idx_followups_due_date ON follow_ups(due_date);
