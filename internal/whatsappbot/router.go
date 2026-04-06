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
	IntentSelection         = "selection"
	IntentConfirmation      = "confirmation"
)

// ParseIntent attempts to determine the user's intent from their free-text input.
func ParseIntent(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))

	// Numeric selections (1-5)
	if len(lower) == 1 && lower >= "1" && lower <= "9" {
		return IntentSelection
	}

	// Confirmations
	if containsAny(lower, "yes", "confirm", "ok", "yep", "نعم", "تأكيد", "اوكي", "تم") {
		return IntentConfirmation
	}

	if containsAny(lower, "book", "schedule", "new appointment", "موعد", "حجز", "جديد") {
		return IntentBookAppointment
	}
	if containsAny(lower, "cancel", "الغاء", "إلغاء", "delete", "حذف") {
		return IntentCancelAppointment
	}
	if containsAny(lower, "next", "upcoming", "when", "القادم", "متى", "موعدي") {
		return IntentViewNext
	}
	if containsAny(lower, "report", "result", "pdf", "تقرير", "نتيجة", "نتائج") {
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
