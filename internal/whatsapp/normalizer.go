package whatsapp

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidPhone = errors.New("invalid phone number: must be in E.164 format (+XXXXXXXXXXX)")

	// e164Regex matches valid E.164 numbers: + followed by 8–15 digits.
	e164Regex = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)
)

// NormalizePhone converts a phone number string to E.164 format.
// It strips spaces, dashes, and parentheses. If the number starts with "00"
// it converts to the "+" prefix. Returns ErrInvalidPhone if the result is invalid.
func NormalizePhone(phone string) (string, error) {
	if phone == "" {
		return "", ErrInvalidPhone
	}

	s := strings.TrimSpace(phone)
	s = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(s)

	// 00XXXXXXXXXXX → +XXXXXXXXXXX
	if strings.HasPrefix(s, "00") {
		s = "+" + s[2:]
	}

	if !e164Regex.MatchString(s) {
		return "", ErrInvalidPhone
	}
	return s, nil
}

// IsValidE164 returns true if the string is already a valid E.164 number.
func IsValidE164(phone string) bool {
	_, err := NormalizePhone(phone)
	return err == nil
}
