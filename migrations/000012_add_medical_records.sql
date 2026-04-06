CREATE TABLE medical_records (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    appointment_id UUID REFERENCES appointments(id) ON DELETE SET NULL,
    doctor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    diagnosis TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_medical_records_tenant_id ON medical_records(tenant_id);
CREATE INDEX idx_medical_records_patient_id ON medical_records(patient_id);
CREATE INDEX idx_medical_records_appointment_id ON medical_records(appointment_id);

CREATE TABLE medical_vitals (
    id UUID PRIMARY KEY,
    medical_record_id UUID NOT NULL REFERENCES medical_records(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    value VARCHAR(255) NOT NULL,
    unit VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_medical_vitals_record_id ON medical_vitals(medical_record_id);

CREATE TABLE medical_medications (
    id UUID PRIMARY KEY,
    medical_record_id UUID NOT NULL REFERENCES medical_records(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    dosage VARCHAR(255) NOT NULL,
    frequency VARCHAR(255) NOT NULL,
    duration VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_medical_medications_record_id ON medical_medications(medical_record_id);
