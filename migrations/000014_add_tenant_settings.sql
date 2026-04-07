-- Migration: Create tenant_settings table
-- Stores all configurable settings per tenant, encrypted where sensitive

CREATE TABLE IF NOT EXISTS tenant_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,

    -- General
    clinic_name VARCHAR(255) NOT NULL DEFAULT '',
    subdomain VARCHAR(100) NOT NULL DEFAULT '',
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    language VARCHAR(10) NOT NULL DEFAULT 'en',

    -- Theme
    theme VARCHAR(20) NOT NULL DEFAULT 'system',
    primary_color VARCHAR(20) NOT NULL DEFAULT '#6366f1',
    secondary_color VARCHAR(20) NOT NULL DEFAULT '#8b5cf6',

    -- Notifications
    email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    whatsapp_enabled BOOLEAN NOT NULL DEFAULT FALSE,

    -- AI
    ai_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ai_provider VARCHAR(50) NOT NULL DEFAULT 'none',
    ai_api_key TEXT NOT NULL DEFAULT '',  -- AES-256-GCM encrypted

    -- Integrations — WhatsApp
    whatsapp_provider VARCHAR(50) NOT NULL DEFAULT 'log',
    whatsapp_webhook_secret TEXT NOT NULL DEFAULT '',  -- AES-256-GCM encrypted

    -- Integrations — Email
    email_provider VARCHAR(50) NOT NULL DEFAULT 'log',
    email_from VARCHAR(255) NOT NULL DEFAULT '',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant_id ON tenant_settings(tenant_id);
