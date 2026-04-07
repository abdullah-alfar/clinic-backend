package settings

import (
	"time"

	"github.com/google/uuid"
)

// TenantSettings is the DB model for the tenant_settings table.
// Sensitive fields (AIAPIKey, WhatsAppWebhookSecret) are stored AES-256-GCM encrypted.
type TenantSettings struct {
	ID       uuid.UUID `db:"id"`
	TenantID uuid.UUID `db:"tenant_id"`

	// General
	ClinicName string `db:"clinic_name"`
	Subdomain  string `db:"subdomain"`
	Timezone   string `db:"timezone"`
	Language   string `db:"language"`

	// Theme
	Theme          string `db:"theme"`
	PrimaryColor   string `db:"primary_color"`
	SecondaryColor string `db:"secondary_color"`

	// Notifications
	EmailEnabled    bool `db:"email_enabled"`
	WhatsAppEnabled bool `db:"whatsapp_enabled"`

	// AI
	AIEnabled   bool   `db:"ai_enabled"`
	AIProvider  string `db:"ai_provider"`
	AIAPIKey    string `db:"ai_api_key"` // encrypted at rest

	// WhatsApp integration
	WhatsAppProvider      string `db:"whatsapp_provider"`
	WhatsAppWebhookSecret string `db:"whatsapp_webhook_secret"` // encrypted at rest

	// Email integration
	EmailProvider string `db:"email_provider"`
	EmailFrom     string `db:"email_from"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// DefaultSettings returns a safe default TenantSettings for a given tenant.
func DefaultSettings(tenantID uuid.UUID) *TenantSettings {
	return &TenantSettings{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ClinicName:     "",
		Subdomain:      "",
		Timezone:       "UTC",
		Language:       "en",
		Theme:          "system",
		PrimaryColor:   "#6366f1",
		SecondaryColor: "#8b5cf6",
		EmailEnabled:    false,
		WhatsAppEnabled: false,
		AIEnabled:       false,
		AIProvider:      "none",
		AIAPIKey:        "",
		WhatsAppProvider:      "log",
		WhatsAppWebhookSecret: "",
		EmailProvider: "log",
		EmailFrom:     "",
	}
}
