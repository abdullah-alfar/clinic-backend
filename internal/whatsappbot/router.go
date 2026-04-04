package whatsappbot

import (
	"strings"
)

const (
	IntentBookAppointment   = "book_appointment"
	IntentCancelAppointment = "cancel_appointment"
	IntentViewNext          = "view_next_appointment"
	IntentSendReport        = "send_report"
	IntentHelp              = "help"
	IntentUnknown           = "unknown"
)

// ParseIntent attempts to determine the user's intent from their free-text input.
// This is a naive keyword-based implementation for the MVP.
func ParseIntent(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))

	if containsAny(lower, "book", "schedule", "new appointment", "موعد", "حجز") {
		return IntentBookAppointment
	}
	if containsAny(lower, "cancel", "الغاء", "إلغاء", "delete") {
		return IntentCancelAppointment
	}
	if containsAny(lower, "next", "upcoming", "when", "القادم", "متى") {
		return IntentViewNext
	}
	if containsAny(lower, "report", "result", "pdf", "تقرير", "نتيجة") {
		return IntentSendReport
	}
	if containsAny(lower, "help", "hi", "hello", "menu", "مساعدة", "مرحبا", "أهلا", "قائمة") {
		return IntentHelp
	}

	return IntentUnknown
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}
