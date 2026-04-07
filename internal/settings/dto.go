package settings

// GetSettingsResponse is the safe API response for GET /settings.
// Sensitive fields are masked — never returned in plaintext.
type GetSettingsResponse struct {
	// General
	ClinicName string `json:"clinic_name"`
	Subdomain  string `json:"subdomain"`
	Timezone   string `json:"timezone"`
	Language   string `json:"language"`

	// Theme
	Theme          string `json:"theme"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`

	// Notifications
	EmailEnabled    bool `json:"email_enabled"`
	WhatsAppEnabled bool `json:"whatsapp_enabled"`

	// AI — api key is masked, never returned
	AIEnabled      bool   `json:"ai_enabled"`
	AIProvider     string `json:"ai_provider"`
	AIAPIKeyIsSet  bool   `json:"ai_api_key_is_set"` // true if an encrypted key exists

	// WhatsApp integration
	WhatsAppProvider             string `json:"whatsapp_provider"`
	WhatsAppWebhookSecretIsSet   bool   `json:"whatsapp_webhook_secret_is_set"`

	// Email integration
	EmailProvider string `json:"email_provider"`
	EmailFrom     string `json:"email_from"`
}

// UpdateSettingsRequest is the payload for PUT /settings.
// Sensitive fields are optional — if empty, the existing value is preserved.
type UpdateSettingsRequest struct {
	// General
	ClinicName string `json:"clinic_name"`
	Subdomain  string `json:"subdomain"`
	Timezone   string `json:"timezone"`
	Language   string `json:"language"`

	// Theme
	Theme          string `json:"theme"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`

	// Notifications
	EmailEnabled    bool `json:"email_enabled"`
	WhatsAppEnabled bool `json:"whatsapp_enabled"`

	// AI — omit or set empty to preserve existing key
	AIEnabled  bool   `json:"ai_enabled"`
	AIProvider string `json:"ai_provider"`
	AIAPIKey   string `json:"ai_api_key,omitempty"` // plaintext, only send if changing

	// WhatsApp integration
	WhatsAppProvider      string `json:"whatsapp_provider"`
	WhatsAppWebhookSecret string `json:"whatsapp_webhook_secret,omitempty"` // only send if changing

	// Email integration
	EmailProvider string `json:"email_provider"`
	EmailFrom     string `json:"email_from"`
}

// TestAIRequest is the payload for POST /settings/test-ai.
type TestAIRequest struct {
	Prompt string `json:"prompt"`
}

// TestAIResponse is the response from POST /settings/test-ai.
type TestAIResponse struct {
	Response string `json:"response"`
	Provider string `json:"provider"`
}

// TestEmailRequest is the payload for POST /settings/test-email.
type TestEmailRequest struct {
	To string `json:"to"`
}

// TestWhatsAppRequest is the payload for POST /settings/test-whatsapp.
type TestWhatsAppRequest struct {
	To string `json:"to"`
}
