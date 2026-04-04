-- Migration: 000010_add_recurrence_rules.sql
-- Adds support for recurring appointments by tracking the template/rule.

CREATE TYPE recurrence_frequency AS ENUM ('weekly', 'monthly');

CREATE TABLE recurrence_rules (
    id              UUID                PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID                NOT NULL REFERENCES tenants(id)     ON DELETE CASCADE,
    patient_id      UUID                NOT NULL REFERENCES patients(id)    ON DELETE CASCADE,
    doctor_id       UUID                NOT NULL REFERENCES doctors(id)    ON DELETE CASCADE,
    frequency       recurrence_frequency NOT NULL,
    interval        INT                 NOT NULL DEFAULT 1,
    day_of_week     SMALLINT            CHECK (day_of_week BETWEEN 0 AND 6),  -- 0=Sun, 6=Sat (for weekly)
    day_of_month    SMALLINT            CHECK (day_of_month BETWEEN 1 AND 31), -- (for monthly)
    start_time      TIME                NOT NULL,
    end_time        TIME                NOT NULL,
    start_date      DATE                NOT NULL,
    end_date        DATE                NOT NULL,
    reason          TEXT,
    status          VARCHAR(20)         NOT NULL DEFAULT 'active', -- active, completed, cancelled
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    CONSTRAINT      chk_recurrence_dates CHECK (end_date >= start_date),
    CONSTRAINT      chk_recurrence_times CHECK (end_time > start_time)
);

-- Add a link from appointments to their parent recurrence rule
ALTER TABLE appointments ADD COLUMN recurrence_rule_id UUID REFERENCES recurrence_rules(id) ON DELETE SET NULL;

CREATE INDEX idx_recurrence_rules_tenant_doctor ON recurrence_rules(tenant_id, doctor_id);
CREATE INDEX idx_recurrence_rules_tenant_patient ON recurrence_rules(tenant_id, patient_id);
CREATE INDEX idx_appointments_recurrence_rule ON appointments(recurrence_rule_id);
