package ops_intelligence

import (
	"github.com/google/uuid"
)

type CommunicationResponse struct {
	ID        uuid.UUID             `json:"id"`
	PatientID uuid.UUID             `json:"patient_id"`
	PatientName string              `json:"patient_name"`
	Channel   CommunicationChannel  `json:"channel"`
	Direction CommunicationDirection `json:"direction"`
	Message   string                `json:"message"`
	Status    string                `json:"status"`
	Priority  CommunicationPriority `json:"priority"`
	Category  string                `json:"category"`
	CreatedAt string                `json:"created_at"`
}

type NoShowRiskResponse struct {
	AppointmentID uuid.UUID       `json:"appointment_id"`
	RiskScore     float64         `json:"risk_score"`
	RiskLevel     NoShowRiskLevel `json:"risk_level"`
	Factors       []string        `json:"factors"`
}

type MissingRevenueResponse struct {
	AppointmentID   uuid.UUID `json:"appointment_id"`
	MissingServices []string  `json:"missing_services"`
}
