package ops_intelligence

import (
	"strings"
)

type Inbox interface {
	Classify(message string) (string, CommunicationPriority)
}

type defaultInbox struct{}

func NewInbox() Inbox {
	return &defaultInbox{}
}

func (i *defaultInbox) Classify(message string) (string, CommunicationPriority) {
	lowerMsg := strings.ToLower(message)

	if strings.Contains(lowerMsg, "emergency") || strings.Contains(lowerMsg, "pain") || strings.Contains(lowerMsg, "bleeding") {
		return "emergency", PriorityUrgent
	}

	if strings.Contains(lowerMsg, "appointment") || strings.Contains(lowerMsg, "book") || strings.Contains(lowerMsg, "schedule") {
		return "booking request", PriorityHigh
	}

	if strings.Contains(lowerMsg, "complaint") || strings.Contains(lowerMsg, "issue") || strings.Contains(lowerMsg, "bad") {
		return "complaint", PriorityMedium
	}

	return "general inquiry" , PriorityLow
}
