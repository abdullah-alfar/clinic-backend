package settings

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when no settings row exists for a tenant.
var ErrNotFound = errors.New("settings: not found")

// Repository defines the data access contract for tenant settings.
type Repository interface {
	GetByTenantID(tenantID uuid.UUID) (*TenantSettings, error)
	Upsert(s *TenantSettings) error
}

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a PostgreSQL-backed settings repository.
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetByTenantID(tenantID uuid.UUID) (*TenantSettings, error) {
	query := `
		SELECT
			id, tenant_id, clinic_name, subdomain, timezone, language,
			theme, primary_color, secondary_color,
			email_enabled, whatsapp_enabled,
			ai_enabled, ai_provider, ai_api_key,
			whatsapp_provider, whatsapp_webhook_secret,
			email_provider, email_from,
			created_at, updated_at
		FROM tenant_settings
		WHERE tenant_id = $1
	`
	s := &TenantSettings{}
	err := r.db.QueryRow(query, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ClinicName, &s.Subdomain, &s.Timezone, &s.Language,
		&s.Theme, &s.PrimaryColor, &s.SecondaryColor,
		&s.EmailEnabled, &s.WhatsAppEnabled,
		&s.AIEnabled, &s.AIProvider, &s.AIAPIKey,
		&s.WhatsAppProvider, &s.WhatsAppWebhookSecret,
		&s.EmailProvider, &s.EmailFrom,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *postgresRepository) Upsert(s *TenantSettings) error {
	s.UpdatedAt = time.Now()
	query := `
		INSERT INTO tenant_settings (
			id, tenant_id, clinic_name, subdomain, timezone, language,
			theme, primary_color, secondary_color,
			email_enabled, whatsapp_enabled,
			ai_enabled, ai_provider, ai_api_key,
			whatsapp_provider, whatsapp_webhook_secret,
			email_provider, email_from,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, NOW(), $19
		)
		ON CONFLICT (tenant_id) DO UPDATE SET
			clinic_name             = EXCLUDED.clinic_name,
			subdomain               = EXCLUDED.subdomain,
			timezone                = EXCLUDED.timezone,
			language                = EXCLUDED.language,
			theme                   = EXCLUDED.theme,
			primary_color           = EXCLUDED.primary_color,
			secondary_color         = EXCLUDED.secondary_color,
			email_enabled           = EXCLUDED.email_enabled,
			whatsapp_enabled        = EXCLUDED.whatsapp_enabled,
			ai_enabled              = EXCLUDED.ai_enabled,
			ai_provider             = EXCLUDED.ai_provider,
			ai_api_key              = EXCLUDED.ai_api_key,
			whatsapp_provider       = EXCLUDED.whatsapp_provider,
			whatsapp_webhook_secret = EXCLUDED.whatsapp_webhook_secret,
			email_provider          = EXCLUDED.email_provider,
			email_from              = EXCLUDED.email_from,
			updated_at              = EXCLUDED.updated_at
	`
	_, err := r.db.Exec(query,
		s.ID, s.TenantID, s.ClinicName, s.Subdomain, s.Timezone, s.Language,
		s.Theme, s.PrimaryColor, s.SecondaryColor,
		s.EmailEnabled, s.WhatsAppEnabled,
		s.AIEnabled, s.AIProvider, s.AIAPIKey,
		s.WhatsAppProvider, s.WhatsAppWebhookSecret,
		s.EmailProvider, s.EmailFrom,
		s.UpdatedAt,
	)
	return err
}
