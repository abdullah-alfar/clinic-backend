package ops_intelligence

import (
	"time"

	"github.com/google/uuid"
)

type CommunicationChannel string

const (
	ChannelWhatsApp CommunicationChannel = "whatsapp"
	ChannelEmail    CommunicationChannel = "email"
	ChannelSMS      CommunicationChannel = "sms"
)

type CommunicationDirection string

const (
	DirectionInbound  CommunicationDirection = "inbound"
	DirectionOutbound CommunicationDirection = "outbound"
)

type CommunicationPriority string

const (
	PriorityLow    CommunicationPriority = "low"
	PriorityMedium CommunicationPriority = "medium"
	PriorityHigh   CommunicationPriority = "high"
	PriorityUrgent CommunicationPriority = "urgent"
)

type Communication struct {
	ID        uuid.UUID              `json:"id"`
	TenantID  uuid.UUID              `json:"tenant_id"`
	PatientID uuid.UUID              `json:"patient_id"`
	Channel   CommunicationChannel   `json:"channel"`
	Direction CommunicationDirection `json:"direction"`
	Message   string                 `json:"message"`
	Status    string                 `json:"status"` // e.g., "received", "read", "pending"
	Priority  CommunicationPriority  `json:"priority"`
	Category  string                 `json:"category"` // AI Classified: "emergency", "booking request", "complaint", "general inquiry"
	CreatedAt time.Time              `json:"created_at"`
}

type NoShowRiskLevel string

const (
	RiskLow    NoShowRiskLevel = "low"
	RiskMedium NoShowRiskLevel = "medium"
	RiskHigh   NoShowRiskLevel = "high"
)

type NoShowRisk struct {
	AppointmentID uuid.UUID       `json:"appointment_id"`
	RiskScore     float64         `json:"risk_score"`
	RiskLevel     NoShowRiskLevel `json:"risk_level"`
	Factors       []string        `json:"factors"` // Reason for the score
}

type MissingRevenue struct {
	AppointmentID   uuid.UUID `json:"appointment_id"`
	MissingServices []string  `json:"missing_services"`
}
