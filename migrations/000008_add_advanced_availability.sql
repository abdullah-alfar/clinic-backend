-- Migration: 000008_add_advanced_availability.sql
-- Adds the advanced doctor availability tables:
--   doctor_schedule   – recurring weekly shifts (replaces/extends doctor_availability)
--   doctor_breaks     – recurring break windows within a shift
--   doctor_exceptions – date-specific overrides (day-off or custom hours)
--
-- The legacy `doctor_availability` table is left intact so existing
-- appointment booking logic continues to work during a rollout.
-- A future migration can backfill doctor_schedule from doctor_availability
-- and drop the old table once all consumers are migrated.

-- ─── doctor_schedule ──────────────────────────────────────────────────────────
-- Stores the recurring weekly working windows of a doctor.
-- Multiple rows per (doctor, day_of_week) are allowed to support
-- multi-shift days (e.g. 08:00-12:00 and 16:00-20:00) without schema changes.

CREATE TABLE doctor_schedule (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id)  ON DELETE CASCADE,
    doctor_id   UUID        NOT NULL REFERENCES doctors(id)  ON DELETE CASCADE,
    day_of_week SMALLINT    NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time  TIME        NOT NULL,
    end_time    TIME        NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_schedule_time CHECK (end_time > start_time)
);

CREATE INDEX idx_doctor_schedule_tenant_doctor_day
    ON doctor_schedule(tenant_id, doctor_id, day_of_week);

-- ─── doctor_breaks ────────────────────────────────────────────────────────────
-- Stores recurring break windows that are carved out of a shift.
-- Linked to doctor_schedule so breaks are cascade-deleted with the parent shift.
-- day_of_week is denormalised here for efficient per-day break lookups during
-- slot generation without requiring an extra JOIN.

CREATE TABLE doctor_breaks (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id)      ON DELETE CASCADE,
    doctor_id   UUID        NOT NULL REFERENCES doctors(id)      ON DELETE CASCADE,
    schedule_id UUID        NOT NULL REFERENCES doctor_schedule(id) ON DELETE CASCADE,
    day_of_week SMALLINT    NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time  TIME        NOT NULL,
    end_time    TIME        NOT NULL,
    label       VARCHAR(100) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_break_time CHECK (end_time > start_time)
);

CREATE INDEX idx_doctor_breaks_tenant_doctor_day
    ON doctor_breaks(tenant_id, doctor_id, day_of_week);

-- ─── doctor_exceptions ────────────────────────────────────────────────────────
-- Stores date-specific overrides that supersede the weekly schedule.
-- type = 'day_off'  → the doctor is fully unavailable on that date.
-- type = 'override' → the doctor works custom hours (start_time/end_time required).
-- The unique constraint on (tenant_id, doctor_id, date) enforces at most one
-- exception per doctor per date, preventing ambiguous availability scenarios.

CREATE TYPE availability_exception_type AS ENUM ('day_off', 'override');

CREATE TABLE doctor_exceptions (
    id          UUID                        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID                        NOT NULL REFERENCES tenants(id)  ON DELETE CASCADE,
    doctor_id   UUID                        NOT NULL REFERENCES doctors(id)  ON DELETE CASCADE,
    date        DATE                        NOT NULL,
    type        availability_exception_type NOT NULL,
    start_time  TIME,        -- NULL when type = 'day_off'
    end_time    TIME,        -- NULL when type = 'day_off'
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT  chk_exception_time CHECK (
        type = 'day_off' OR (start_time IS NOT NULL AND end_time IS NOT NULL AND end_time > start_time)
    ),
    CONSTRAINT  uq_doctor_exception_date UNIQUE (tenant_id, doctor_id, date)
);

CREATE INDEX idx_doctor_exceptions_tenant_doctor_date
    ON doctor_exceptions(tenant_id, doctor_id, date);
