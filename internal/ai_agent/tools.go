package ai_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/availability"
	"clinic-backend/internal/patient"
	"clinic-backend/internal/search"

	"github.com/google/uuid"
)

// AITool defines an action the agent can perform.
type AITool interface {
	Name() string
	Description() string
	RequiresConfirmation() bool
	// Execute performs the read action or prepares the write action payload
	Execute(ctx context.Context, tenantID, patientID string, input json.RawMessage) (string, error)
	// ExecuteConfirmed actually mutates the state after user confirmation
	ExecuteConfirmed(ctx context.Context, tenantID string, payload json.RawMessage) (string, error)
}

type SystemTools struct {
	tools map[string]AITool
}

func NewSystemTools() *SystemTools {
	return &SystemTools{
		tools: make(map[string]AITool),
	}
}

func (s *SystemTools) Register(tool AITool) {
	s.tools[tool.Name()] = tool
}

func (s *SystemTools) GetTool(name string) (AITool, bool) {
	tool, exists := s.tools[name]
	return tool, exists
}

func (s *SystemTools) GetAllTools() []AITool {
	var list []AITool
	for _, t := range s.tools {
		list = append(list, t)
	}
	return list
}

// ---------------------------------------------------------
// Global Search Tool (Read Only)
// ---------------------------------------------------------

type globalSearchInput struct {
	Query string `json:"query"`
}

type GlobalSearchTool struct {
	searchSvc search.SearchService
}

func NewGlobalSearchTool(svc search.SearchService) *GlobalSearchTool {
	return &GlobalSearchTool{searchSvc: svc}
}
func (t *GlobalSearchTool) Name() string { return "GlobalSearch" }
func (t *GlobalSearchTool) Description() string {
	return "Searches across the entire system for patients, appointments, invoices, doctors, etc. Input: {\"query\": \"search text\"}"
}
func (t *GlobalSearchTool) RequiresConfirmation() bool { return false }
func (t *GlobalSearchTool) Execute(ctx context.Context, tenantID, patientID string, input json.RawMessage) (string, error) {
	var args globalSearchInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}

	results, err := t.searchSvc.GlobalSearch(ctx, search.SearchRequest{Query: args.Query})
	if err != nil {
		return "", err
	}

	b, _ := json.MarshalIndent(results, "", "  ")
	return string(b), nil
}
func (t *GlobalSearchTool) ExecuteConfirmed(ctx context.Context, tenantID string, payload json.RawMessage) (string, error) {
	return "", ErrUnauthorized
}

// ---------------------------------------------------------
// Check Availability Tool (Read Only)
// ---------------------------------------------------------

type checkAvailabilityInput struct {
	DoctorID string `json:"doctor_id"`
	Date     string `json:"date"` // YYYY-MM-DD
}

type GetDoctorAvailabilityTool struct {
	availSvc *availability.AvailabilityService
}

func NewGetDoctorAvailabilityTool(svc *availability.AvailabilityService) *GetDoctorAvailabilityTool {
	return &GetDoctorAvailabilityTool{availSvc: svc}
}
func (t *GetDoctorAvailabilityTool) Name() string { return "GetDoctorAvailability" }
func (t *GetDoctorAvailabilityTool) Description() string {
	return "Checks a doctor's availability on a specific date. Input: {\"doctor_id\": \"uuid\", \"date\": \"2023-10-25\"}"
}
func (t *GetDoctorAvailabilityTool) RequiresConfirmation() bool { return false }
func (t *GetDoctorAvailabilityTool) Execute(ctx context.Context, tenantID, patientID string, input json.RawMessage) (string, error) {
	var args checkAvailabilityInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %v", err)
	}
	
	date, err := time.Parse("2006-01-02", args.Date)
	if err != nil {
		return "", fmt.Errorf("invalid date format, use YYYY-MM-DD")
	}

	tID, _ := uuid.Parse(tenantID)
	dID, _ := uuid.Parse(args.DoctorID)

	slots, err := t.availSvc.GetAvailableSlots(ctx, tID, dID, availability.SlotQueryParams{
		DateFrom: date,
		DateTo:   date,
	})
	if err != nil {
		return "", err
	}

	b, _ := json.MarshalIndent(slots, "", "  ")
	return string(b), nil
}
func (t *GetDoctorAvailabilityTool) ExecuteConfirmed(ctx context.Context, tenantID string, payload json.RawMessage) (string, error) {
	return "", ErrUnauthorized
}

// ---------------------------------------------------------
// Create Patient Tool (Write)
// ---------------------------------------------------------

type CreatePatientTool struct {
	patientSvc *patient.PatientService
}

func NewCreatePatientTool(svc *patient.PatientService) *CreatePatientTool {
	return &CreatePatientTool{patientSvc: svc}
}
func (t *CreatePatientTool) Name() string { return "CreatePatient" }
func (t *CreatePatientTool) Description() string {
	return "Creates a new patient. Requires confirmation. Input: {\"first_name\": \"...\", \"last_name\": \"...\", \"phone\": \"...\", \"email\": \"...\"}"
}
func (t *CreatePatientTool) RequiresConfirmation() bool { return true }
func (t *CreatePatientTool) Execute(ctx context.Context, tenantID, patientID string, input json.RawMessage) (string, error) {
	return "I have prepared the patient creation. Please confirm.", nil
}
func (t *CreatePatientTool) ExecuteConfirmed(ctx context.Context, tenantID string, payload json.RawMessage) (string, error) {
	var p patient.Patient
	if err := json.Unmarshal(payload, &p); err != nil {
		return "", err
	}
	tID, _ := uuid.Parse(tenantID)
	p.TenantID = tID
	
	err := t.patientSvc.CreatePatient(&p, uuid.Nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Patient created successfully with ID: %s", p.ID), nil
}

// ---------------------------------------------------------
// Book Appointment Tool (Write)
// ---------------------------------------------------------

type bookAppointmentInput struct {
	DoctorID  string `json:"doctor_id"`
	PatientID string `json:"patient_id"`
	StartTime string `json:"start_time"` // RFC3339
	Duration  int    `json:"duration"`   // Minutes
	Type      string `json:"type"`       // consultation, followup
}

type BookAppointmentTool struct {
	apptSvc *appointment.AppointmentService
}

func NewBookAppointmentTool(svc *appointment.AppointmentService) *BookAppointmentTool {
	return &BookAppointmentTool{apptSvc: svc}
}
func (t *BookAppointmentTool) Name() string { return "BookAppointment" }
func (t *BookAppointmentTool) Description() string {
	return "Books a new appointment. Requires confirmation. Input: {\"doctor_id\": \"uuid\", \"patient_id\": \"uuid\", \"start_time\": \"RFC3339\", \"duration\": 30, \"type\": \"consultation\"}"
}
func (t *BookAppointmentTool) RequiresConfirmation() bool { return true }
func (t *BookAppointmentTool) Execute(ctx context.Context, tenantID, patientID string, input json.RawMessage) (string, error) {
	return "I have prepared the appointment booking. Please confirm.", nil
}
func (t *BookAppointmentTool) ExecuteConfirmed(ctx context.Context, tenantID string, payload json.RawMessage) (string, error) {
	var args bookAppointmentInput
	if err := json.Unmarshal(payload, &args); err != nil {
		return "", err
	}

	start, err := time.Parse(time.RFC3339, args.StartTime)
	if err != nil {
		return "", err
	}

	tID, _ := uuid.Parse(tenantID)
	dID, _ := uuid.Parse(args.DoctorID)
	pID, _ := uuid.Parse(args.PatientID)

	appt, err := t.apptSvc.ScheduleAppointment(tID, pID, dID, start, start.Add(time.Duration(args.Duration)*time.Minute), uuid.Nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Appointment booked successfully with ID: %s", appt.ID), nil
}

