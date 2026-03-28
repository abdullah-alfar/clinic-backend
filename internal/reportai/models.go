package reportai

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ReportAIAnalysis struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	PatientID      uuid.UUID       `json:"patient_id"`
	AttachmentID   uuid.UUID       `json:"attachment_id"`
	AnalysisType   string          `json:"analysis_type"` // e.g., 'summary', 'extraction'
	Status         string          `json:"status"`        // 'pending', 'completed', 'failed'
	Summary        *string         `json:"summary,omitempty"`
	StructuredData json.RawMessage `json:"structured_data,omitempty"`
	RawResponse    json.RawMessage `json:"-"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
