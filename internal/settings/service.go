package settings

import (
	"context"
	"errors"
	"fmt"

	aipkg "clinic-backend/internal/ai"
	"clinic-backend/internal/mail"
	"clinic-backend/internal/whatsapp"

	"github.com/google/uuid"
)

// Service contains all business logic for tenant settings.
type Service struct {
	repo Repository
}

// NewService returns a new settings Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetSettings fetches settings for a tenant, returning defaults if none exist.
// Sensitive fields are masked in the response.
func (s *Service) GetSettings(tenantID uuid.UUID) (*GetSettingsResponse, error) {
	settings, err := s.repo.GetByTenantID(tenantID)
	if errors.Is(err, ErrNotFound) {
		settings = DefaultSettings(tenantID)
	} else if err != nil {
		return nil, fmt.Errorf("settings.GetSettings: %w", err)
	}

	return toResponse(settings), nil
}

// UpdateSettings validates, encrypts secrets, and persists updated settings.
// If a sensitive field (AIAPIKey, WebhookSecret) is empty in the request,
// the existing encrypted value is preserved.
func (s *Service) UpdateSettings(tenantID uuid.UUID, req UpdateSettingsRequest) error {
	// Load existing record (or get defaults)
	existing, err := s.repo.GetByTenantID(tenantID)
	if errors.Is(err, ErrNotFound) {
		existing = DefaultSettings(tenantID)
	} else if err != nil {
		return fmt.Errorf("settings.UpdateSettings: fetch existing: %w", err)
	}

	// Apply non-sensitive fields
	existing.ClinicName      = req.ClinicName
	existing.Subdomain       = req.Subdomain
	existing.Timezone        = req.Timezone
	existing.Language        = req.Language
	existing.Theme           = req.Theme
	existing.PrimaryColor    = req.PrimaryColor
	existing.SecondaryColor  = req.SecondaryColor
	existing.EmailEnabled    = req.EmailEnabled
	existing.WhatsAppEnabled = req.WhatsAppEnabled
	existing.AIEnabled       = req.AIEnabled
	existing.AIProvider      = req.AIProvider
	existing.WhatsAppProvider = req.WhatsAppProvider
	existing.EmailProvider   = req.EmailProvider
	existing.EmailFrom       = req.EmailFrom

	// Only update encrypted fields if caller provided new values
	if req.AIAPIKey != "" {
		encrypted, err := Encrypt(req.AIAPIKey)
		if err != nil {
			return fmt.Errorf("settings.UpdateSettings: encrypt ai key: %w", err)
		}
		existing.AIAPIKey = encrypted
	}
	if req.WhatsAppWebhookSecret != "" {
		encrypted, err := Encrypt(req.WhatsAppWebhookSecret)
		if err != nil {
			return fmt.Errorf("settings.UpdateSettings: encrypt webhook secret: %w", err)
		}
		existing.WhatsAppWebhookSecret = encrypted
	}

	return s.repo.Upsert(existing)
}

// TestAI creates a live AI provider from tenant settings and calls Generate().
func (s *Service) TestAI(tenantID uuid.UUID, prompt string) (string, error) {
	settings, err := s.repo.GetByTenantID(tenantID)
	if err != nil {
		return "", fmt.Errorf("settings.TestAI: load settings: %w", err)
	}
	if !settings.AIEnabled {
		return "", fmt.Errorf("AI is disabled in settings")
	}

	apiKey, err := Decrypt(settings.AIAPIKey)
	if err != nil {
		return "", fmt.Errorf("settings.TestAI: decrypt api key: %w", err)
	}

	provider, err := aipkg.NewProvider(settings.AIProvider, apiKey)
	if err != nil {
		return "", fmt.Errorf("settings.TestAI: create provider: %w", err)
	}

	return provider.Generate(context.Background(), prompt)
}

// TestEmail sends a test email using the tenant's configured email provider.
func (s *Service) TestEmail(tenantID uuid.UUID, to string) error {
	settings, err := s.repo.GetByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("settings.TestEmail: %w", err)
	}

	sender := buildEmailSender(settings)
	return sender.Send(context.Background(), mail.EmailMessage{
		To:       to,
		From:     settings.EmailFrom,
		Subject:  "Test Email from Clinic System",
		TextBody: "This is a test email sent from your Clinic Settings control panel. If you received this, your email integration is working correctly.",
	})
}

// TestWhatsApp sends a test WhatsApp message using the tenant's configured provider.
func (s *Service) TestWhatsApp(tenantID uuid.UUID, to string) error {
	settings, err := s.repo.GetByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("settings.TestWhatsApp: %w", err)
	}

	sender, err := buildWhatsAppSender(settings)
	if err != nil {
		return fmt.Errorf("settings.TestWhatsApp: build sender: %w", err)
	}

	_, err = sender.Send(context.Background(), whatsapp.WhatsAppMessage{
		To:   to,
		Body: "This is a test message from your Clinic System. Your WhatsApp integration is working correctly! ✅",
	})
	return err
}

// toResponse converts a TenantSettings model to the safe API response,
// masking all sensitive fields.
func toResponse(s *TenantSettings) *GetSettingsResponse {
	return &GetSettingsResponse{
		ClinicName:                   s.ClinicName,
		Subdomain:                    s.Subdomain,
		Timezone:                     s.Timezone,
		Language:                     s.Language,
		Theme:                        s.Theme,
		PrimaryColor:                 s.PrimaryColor,
		SecondaryColor:               s.SecondaryColor,
		EmailEnabled:                 s.EmailEnabled,
		WhatsAppEnabled:              s.WhatsAppEnabled,
		AIEnabled:                    s.AIEnabled,
		AIProvider:                   s.AIProvider,
		AIAPIKeyIsSet:                s.AIAPIKey != "",
		WhatsAppProvider:             s.WhatsAppProvider,
		WhatsAppWebhookSecretIsSet:   s.WhatsAppWebhookSecret != "",
		EmailProvider:                s.EmailProvider,
		EmailFrom:                    s.EmailFrom,
	}
}

// buildEmailSender constructs an EmailSender based on tenant settings.
// Currently returns LogEmailSender; SMTP/Resend/SendGrid can be added here.
func buildEmailSender(s *TenantSettings) mail.EmailSender {
	switch s.EmailProvider {
	// TODO: Add smtp, resend, sendgrid cases when credentials stored
	default:
		return mail.NewLogEmailSender()
	}
}

// buildWhatsAppSender constructs a WhatsAppSender based on tenant settings.
func buildWhatsAppSender(s *TenantSettings) (whatsapp.WhatsAppSender, error) {
	return whatsapp.NewSender(whatsapp.SenderFactoryConfig{
		Provider: s.WhatsAppProvider,
	})
}
