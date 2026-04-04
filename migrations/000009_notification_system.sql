-- Migration: 000009_notification_system.sql
-- Adds tables for the full notification and patient messaging system.

-- ─── ENUMs ────────────────────────────────────────────────────────────────────
CREATE TYPE notification_channel AS ENUM ('email', 'whatsapp', 'in_app');
CREATE TYPE notification_status  AS ENUM ('pending', 'sent', 'failed', 'skipped');
CREATE TYPE notification_event   AS ENUM (
    'appointment_created',
    'appointment_confirmed',
    'appointment_canceled',
    'appointment_rescheduled',
    'appointment_reminder'
);
CREATE TYPE message_direction AS ENUM ('inbound', 'outbound');

-- ─── outbound_notifications ───────────────────────────────────────────────────
-- Records each outbound notification attempt (email, whatsapp) per patient.
-- Separate from the in-app `notifications` table (staff-facing).
CREATE TABLE outbound_notifications (
    id                   UUID                  PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id            UUID                  NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_id           UUID                  REFERENCES patients(id) ON DELETE SET NULL,
    appointment_id       UUID                  REFERENCES appointments(id) ON DELETE SET NULL,
    channel              notification_channel  NOT NULL,
    event_type           notification_event    NOT NULL,
    recipient            VARCHAR(255)          NOT NULL,
    subject              TEXT,
    message              TEXT                  NOT NULL,
    status               notification_status   NOT NULL DEFAULT 'pending',
    provider             VARCHAR(100),
    provider_message_id  VARCHAR(255),
    error_message        TEXT,
    scheduled_for        TIMESTAMPTZ,
    sent_at              TIMESTAMPTZ,
    created_at           TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbound_notif_tenant_patient     ON outbound_notifications(tenant_id, patient_id);
CREATE INDEX idx_outbound_notif_tenant_appointment ON outbound_notifications(tenant_id, appointment_id);
CREATE INDEX idx_outbound_notif_status             ON outbound_notifications(status) WHERE status = 'pending';

-- ─── patient_notification_preferences ────────────────────────────────────────
-- Per-patient opt-in/out per channel and event type.
-- If no row exists, defaults apply (email=true, whatsapp=false).
CREATE TABLE patient_notification_preferences (
    id                              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id                       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_id                      UUID        NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    email_enabled                   BOOLEAN     NOT NULL DEFAULT true,
    whatsapp_enabled                BOOLEAN     NOT NULL DEFAULT false,
    reminder_enabled                BOOLEAN     NOT NULL DEFAULT true,
    appointment_created_enabled     BOOLEAN     NOT NULL DEFAULT true,
    appointment_confirmed_enabled   BOOLEAN     NOT NULL DEFAULT true,
    appointment_canceled_enabled    BOOLEAN     NOT NULL DEFAULT true,
    appointment_rescheduled_enabled BOOLEAN     NOT NULL DEFAULT true,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, patient_id)
);

CREATE INDEX idx_pnp_tenant_patient ON patient_notification_preferences(tenant_id, patient_id);

-- ─── appointment_reminders ────────────────────────────────────────────────────
-- Tracks which channels have had reminders enqueued/sent for an appointment.
-- The unique constraint prevents duplicate reminder scheduling per channel.
CREATE TABLE appointment_reminders (
    id             UUID                 PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID                 NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    appointment_id UUID                 NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
    channel        notification_channel NOT NULL,
    scheduled_for  TIMESTAMPTZ          NOT NULL,
    sent_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    UNIQUE(appointment_id, channel)
);

CREATE INDEX idx_appt_reminders_scheduled ON appointment_reminders(scheduled_for) WHERE sent_at IS NULL;

-- ─── whatsapp_bot_sessions ────────────────────────────────────────────────────
-- Stateful WhatsApp bot conversation session per phone number per tenant.
CREATE TABLE whatsapp_bot_sessions (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_id      UUID         REFERENCES patients(id) ON DELETE SET NULL,
    phone_number    VARCHAR(50)  NOT NULL,
    current_flow    VARCHAR(100) NOT NULL DEFAULT 'menu',
    current_step    VARCHAR(100) NOT NULL DEFAULT 'start',
    state           JSONB        NOT NULL DEFAULT '{}',
    last_message_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, phone_number)
);

CREATE INDEX idx_wbs_tenant_phone ON whatsapp_bot_sessions(tenant_id, phone_number);

-- ─── whatsapp_messages ────────────────────────────────────────────────────────
-- Full inbound/outbound message log for audit and debugging.
CREATE TABLE whatsapp_messages (
    id                  UUID              PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id           UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    patient_id          UUID              REFERENCES patients(id) ON DELETE SET NULL,
    direction           message_direction NOT NULL,
    phone_number        VARCHAR(50)       NOT NULL,
    message_type        VARCHAR(50)       NOT NULL DEFAULT 'text',
    content             TEXT              NOT NULL,
    metadata            JSONB             NOT NULL DEFAULT '{}',
    provider_message_id VARCHAR(255),
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wm_tenant_phone   ON whatsapp_messages(tenant_id, phone_number);
CREATE INDEX idx_wm_tenant_patient ON whatsapp_messages(tenant_id, patient_id);
