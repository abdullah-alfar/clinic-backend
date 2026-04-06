package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	WhatsApp WhatsAppConfig
}

type WhatsAppConfig struct {
	Provider         string
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFrom       string
}

func Load() (*Config, error) {
	cfg := &Config{
		WhatsApp: WhatsAppConfig{
			Provider:         getEnv("WHATSAPP_PROVIDER", "log"),
			TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
			TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
			TwilioFrom:       getEnv("TWILIO_WHATSAPP_FROM", ""),
		},
	}

	if strings.EqualFold(cfg.WhatsApp.Provider, "twilio") {
		if cfg.WhatsApp.TwilioAccountSID == "" {
			return nil, fmt.Errorf("TWILIO_ACCOUNT_SID is required when WHATSAPP_PROVIDER=twilio")
		}
		if cfg.WhatsApp.TwilioAuthToken == "" {
			return nil, fmt.Errorf("TWILIO_AUTH_TOKEN is required when WHATSAPP_PROVIDER=twilio")
		}
		if cfg.WhatsApp.TwilioFrom == "" {
			return nil, fmt.Errorf("TWILIO_WHATSAPP_FROM is required when WHATSAPP_PROVIDER=twilio")
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
