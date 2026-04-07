package ai_core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"clinic-backend/internal/appointment"
	"clinic-backend/internal/scheduling"
	"clinic-backend/internal/search"
)

// Tool represents an atomic action the AI can execute.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]interface{} // JSON Schema representing the input structure
	Execute(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, input json.RawMessage) (string, error)
}

// SystemTools bundles all injected dependencies and maps available tools.
type SystemTools struct {
	tools map[string]Tool
}

func NewSystemTools() *SystemTools {
	return &SystemTools{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (st *SystemTools) Register(t Tool) {
	st.tools[t.Name()] = t
}

func (st *SystemTools) GetTool(name string) (Tool, bool) {
	t, ok := st.tools[name]
	return t, ok
}

func (st *SystemTools) AllTools() []Tool {
	all := make([]Tool, 0, len(st.tools))
	for _, t := range st.tools {
		all = append(all, t)
	}
	return all
}

// -----------------------------------------------------------------------------
// GetAvailableSlotsTool
// -----------------------------------------------------------------------------

type getSlotsInput struct {
	DoctorID *uuid.UUID `json:"doctor_id,omitempty"`
	Date     string     `json:"date"` // RFC 3339 or "YYYY-MM-DD"
}

type GetAvailableSlotsTool struct {
	scheduler *scheduling.SmartSchedulingService
}

func NewGetAvailableSlotsTool(scheduler *scheduling.SmartSchedulingService) *GetAvailableSlotsTool {
	return &GetAvailableSlotsTool{scheduler: scheduler}
}

func (t *GetAvailableSlotsTool) Name() string { return "GetAvailableSlotsTool" }
func (t *GetAvailableSlotsTool) Description() string {
	return "Gets available appointment slots. Requires a specific date. Useful when the user asks 'Are there any slots tomorrow?'"
}
func (t *GetAvailableSlotsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"doctor_id": map[string]interface{}{"type": "string", "description": "Optional UUID of the doctor. Omit if looking for any doctor."},
			"date":      map[string]interface{}{"type": "string", "description": "Required Date in YYYY-MM-DD format based on the user request."},
		},
		"required": []string{"date"},
	}
}

func (t *GetAvailableSlotsTool) Execute(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, input json.RawMessage) (string, error) {
	var in getSlotsInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", err
	}
	
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil {
		// Fallback parse attempt
		if d2, err2 := time.Parse(time.RFC3339, in.Date); err2 == nil {
			d = d2
		} else {
			return "Error: Invalid date format. Must be YYYY-MM-DD.", nil
		}
	}

	startOfDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	suggestions, err := t.scheduler.SuggestSlots(ctx, tenantID, scheduling.SuggestionRequest{
		DateFrom:        startOfDay,
		DateTo:          endOfDay,
		DoctorID:        in.DoctorID,
		DurationMinutes: 30, // Default assume 30m slots
		Strategy:        scheduling.StrategyFastest,
	})

	if err != nil {
		return fmt.Sprintf("Error fetching slots: %v", err), nil
	}

	if len(suggestions) == 0 {
		return "No slots available for this date.", nil
	}

	b, _ := json.Marshal(suggestions)
	return string(b), nil
}

// -----------------------------------------------------------------------------
// SearchPatientsTool
// -----------------------------------------------------------------------------

type searchPatientsInput struct {
	Query string `json:"query"`
}

type SearchPatientsTool struct {
	searchSvc search.SearchService
}

func NewSearchPatientsTool(svc search.SearchService) *SearchPatientsTool {
	return &SearchPatientsTool{searchSvc: svc}
}

func (t *SearchPatientsTool) Name() string { return "SearchPatientsTool" }
func (t *SearchPatientsTool) Description() string {
	return "Search for matching patients by name or phone number. Use when you need to find a patient UUID."
}
func (t *SearchPatientsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "The name or phone number to search."},
		},
		"required": []string{"query"},
	}
}

func (t *SearchPatientsTool) Execute(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, input json.RawMessage) (string, error) {
	var in searchPatientsInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", err
	}

	searchData, err := t.searchSvc.GlobalSearch(ctx, tenantID, in.Query, []string{"patient"})
	if err != nil {
		return fmt.Sprintf("Search error: %v", err), nil
	}
	
	if len(searchData.Groups) == 0 || len(searchData.Groups[0].Results) == 0 {
		return "No matching patients found.", nil
	}
	b, _ := json.Marshal(searchData.Groups[0].Results)
	return string(b), nil
}

// -----------------------------------------------------------------------------
// CreateAppointmentTool
// -----------------------------------------------------------------------------
type createAppointmentInput struct {
	DoctorID *uuid.UUID `json:"doctor_id"`
	Date     string     `json:"date"` // complete start time e.g., 2024-05-15T14:30:00Z
}

type CreateAppointmentTool struct {
	apptSvc *appointment.AppointmentService
}

func NewCreateAppointmentTool(svc *appointment.AppointmentService) *CreateAppointmentTool {
	return &CreateAppointmentTool{apptSvc: svc}
}

func (t *CreateAppointmentTool) Name() string { return "CreateAppointmentTool" }
func (t *CreateAppointmentTool) Description() string {
	return "Books a new appointment. Requires a target doctor_id, start date/time, and the session MUST have an active patient context."
}
func (t *CreateAppointmentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"doctor_id": map[string]interface{}{"type": "string"},
			"date":      map[string]interface{}{"type": "string"},
		},
		"required": []string{"doctor_id", "date"},
	}
}

func (t *CreateAppointmentTool) Execute(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, input json.RawMessage) (string, error) {
	if patientID == nil {
		return "Error: Cannot book appointment because the current session does not have an identified patient. Ask the user to clarify who they are or log in.", nil
	}

	var in createAppointmentInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", err
	}

	if in.DoctorID == nil {
		return "Error: doctor_id is required. Run GetAvailableSlotsTool first to find a valid doctor_id.", nil
	}

	start, err := time.Parse(time.RFC3339, in.Date)
	if err != nil {
		return "Error: Invalid date format. Use RFC3339.", nil
	}
	end := start.Add(30 * time.Minute)

	// Since AI System is booking it on behalf, act as system or patient
	appt, err := t.apptSvc.ScheduleAppointment(tenantID, *patientID, *in.DoctorID, start, end, *patientID)
	if err != nil {
		return fmt.Sprintf("Failed to book: %v", err), nil
	}
	
	b, _ := json.Marshal(appt)
	return fmt.Sprintf("Appointment successfully booked: %s", string(b)), nil
}

// -----------------------------------------------------------------------------
// CancelAppointmentTool 
// -----------------------------------------------------------------------------
type cancelAppointmentInput struct {
	AppointmentID uuid.UUID `json:"appointment_id"`
}

type CancelAppointmentTool struct {
	apptSvc *appointment.AppointmentService
}

func NewCancelAppointmentTool(svc *appointment.AppointmentService) *CancelAppointmentTool {
	return &CancelAppointmentTool{apptSvc: svc}
}

func (t *CancelAppointmentTool) Name() string { return "CancelAppointmentTool" }
func (t *CancelAppointmentTool) Description() string {
	return "Cancels an upcoming appointment."
}
func (t *CancelAppointmentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"appointment_id": map[string]interface{}{"type": "string"},
		},
		"required": []string{"appointment_id"},
	}
}

func (t *CancelAppointmentTool) Execute(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID, input json.RawMessage) (string, error) {
	var in cancelAppointmentInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", err
	}

	err := t.apptSvc.UpdateStatus(tenantID, in.AppointmentID, "canceled", uuid.Nil)
	if err != nil {
		return fmt.Sprintf("Error canceling appointment: %v", err), nil
	}
	return "Appointment successfully canceled.", nil
}

// Additional abstract tools (Medical Insights, Search Appointments, etc.) can be similarly injected.
