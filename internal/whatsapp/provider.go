package whatsapp

import (
	"fmt"
	"strings"
)

type Provider string

const (
	ProviderLog    Provider = "log"
	ProviderMeta   Provider = "meta"
	ProviderTwilio Provider = "twilio"
)

type SenderFactoryConfig struct {
	Provider string

	// Twilio
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFrom       string // whatsapp:+14155238886

	// Meta
	MetaPhoneNumberID string
	MetaAccessToken   string
}

func NewSender(cfg SenderFactoryConfig) (WhatsAppSender, error) {
	provider := Provider(strings.ToLower(strings.TrimSpace(cfg.Provider)))

	switch provider {
	case "", ProviderLog:
		return NewLogWhatsAppSender(), nil

	case ProviderTwilio:
		if strings.TrimSpace(cfg.TwilioAccountSID) == "" {
			return nil, fmt.Errorf("missing twilio account sid")
		}
		if strings.TrimSpace(cfg.TwilioAuthToken) == "" {
			return nil, fmt.Errorf("missing twilio auth token")
		}
		if strings.TrimSpace(cfg.TwilioFrom) == "" {
			return nil, fmt.Errorf("missing twilio from number")
		}

		return NewTwilioWhatsAppSender(
			cfg.TwilioAccountSID,
			cfg.TwilioAuthToken,
			cfg.TwilioFrom,
		), nil

	case ProviderMeta:
		if strings.TrimSpace(cfg.MetaPhoneNumberID) == "" {
			return nil, fmt.Errorf("missing meta phone number id")
		}
		if strings.TrimSpace(cfg.MetaAccessToken) == "" {
			return nil, fmt.Errorf("missing meta access token")
		}

		return NewMetaWhatsAppSender(
			cfg.MetaPhoneNumberID,
			cfg.MetaAccessToken,
		), nil

	default:
		return nil, fmt.Errorf("unsupported whatsapp provider: %s", cfg.Provider)
	}
}
