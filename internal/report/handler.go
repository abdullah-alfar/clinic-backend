package report

import (
	"net/http"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type ReportHandler struct {
	svc *ReportService
}

func NewReportHandler(svc *ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, _ := shared.GetUserContext(r.Context())
	
	summary, err := h.svc.GetSummary(userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to build summary", "REPORT_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, summary, "success")
}

func (h *ReportHandler) HandleAppointmentsReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, _ := shared.GetUserContext(r.Context())
	
	q := r.URL.Query()
	status := q.Get("status")
	
	var docID *uuid.UUID
	if d := q.Get("doctor_id"); d != "" {
		parsed, err := uuid.Parse(d)
		if err == nil {
			docID = &parsed
		}
	}

	var dFrom *time.Time
	if d := q.Get("date_from"); d != "" {
		parsed, err := time.Parse(time.RFC3339, d)
		if err == nil {
			dFrom = &parsed
		}
	}

	var dTo *time.Time
	if d := q.Get("date_to"); d != "" {
		parsed, err := time.Parse(time.RFC3339, d)
		if err == nil {
			dTo = &parsed
		}
	}

	appts, err := h.svc.GetAppointmentsReport(userCtx.TenantID, docID, status, dFrom, dTo)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to report appointments", "REPORT_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, appts, "success")
}

func (h *ReportHandler) HandlePatientsReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, _ := shared.GetUserContext(r.Context())
	q := r.URL.Query()

	var dFrom *time.Time
	if d := q.Get("date_from"); d != "" {
		parsed, err := time.Parse(time.RFC3339, d)
		if err == nil {
			dFrom = &parsed
		}
	}

	var dTo *time.Time
	if d := q.Get("date_to"); d != "" {
		parsed, err := time.Parse(time.RFC3339, d)
		if err == nil {
			dTo = &parsed
		}
	}

	patients, err := h.svc.GetPatientsReport(userCtx.TenantID, dFrom, dTo)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to report patients", "REPORT_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, patients, "success")
}
