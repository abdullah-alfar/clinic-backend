package medical

import (
	"github.com/google/uuid"
)

type CreateMedicalRecordRequest struct {
	AppointmentID *uuid.UUID                       `json:"appointment_id"`
	Diagnosis     string                           `json:"diagnosis"`
	Notes         string                           `json:"notes"`
	Vitals        []CreateMedicalVitalRequest      `json:"vitals"`
	Medications   []CreateMedicalMedicationRequest `json:"medications"`
}

type AddProcedureReq struct {
	ProcedureCatalogID uuid.UUID `json:"procedure_catalog_id"`
	Notes              *string   `json:"notes"`
}

type UpdateMedicalRecordRequest struct {
	AppointmentID *uuid.UUID                       `json:"appointment_id"`
	Diagnosis     *string                          `json:"diagnosis"`
	Notes         *string                          `json:"notes"`
	Vitals        []CreateMedicalVitalRequest      `json:"vitals"` // Handled as full-replacement
	Medications   []CreateMedicalMedicationRequest `json:"medications"`
}

type CreateMedicalVitalRequest struct {
	Type  string  `json:"type"`
	Value string  `json:"value"`
	Unit  *string `json:"unit"`
}

type CreateMedicalMedicationRequest struct {
	Name      string  `json:"name"`
	Dosage    string  `json:"dosage"`
	Frequency string  `json:"frequency"`
	Duration  *string `json:"duration"`
	Notes     *string `json:"notes"`
}

type MedicalRecordResponse struct {
	Record      *MedicalRecord            `json:"record"`
	Vitals      []*MedicalVital           `json:"vitals"`
	Medications []*MedicalMedication      `json:"medications"`
	Procedures  []*MedicalRecordProcedure `json:"procedures"`
}
