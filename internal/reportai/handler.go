package reportai

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"clinic-backend/internal/attachment"
)

type ReportAIHandler struct {
	svc     *ReportAIService
	attRepo attachment.Repository
}

// We inject the attachment.Repository to retrieve the attachment's file URL before sending to AI
func NewReportAIHandler(svc *ReportAIService, attRepo attachment.Repository) *ReportAIHandler {
	return &ReportAIHandler{
		svc:     svc,
		attRepo: attRepo,
	}
}

// HandleAnalyzeReport: POST /api/v1/attachments/{id}/analyze
func (h *ReportAIHandler) HandleAnalyzeReport(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	attIDStr := r.PathValue("id")
	attID, err := uuid.Parse(attIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid attachment ID", "BAD_REQUEST", nil)
		return
	}

	var req AnalyzeReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid JSON payload", "BAD_REQUEST", err.Error())
		return
	}
	if req.AnalysisType == "" {
		req.AnalysisType = "summary"
	}

	// Verify attachment exists and belongs to tenant
	att, err := h.attRepo.GetByID(userCtx.TenantID, attID)
	if err != nil || att == nil {
		myhttp.RespondError(w, http.StatusNotFound, "attachment not found", "NOT_FOUND", nil)
		return
	}

	analysis, err := h.svc.RequestAnalysis(
		userCtx.TenantID, 
		att.PatientID, 
		att.ID, 
		userCtx.UserID, 
		att.FileURL, 
		att.MimeType, 
		req.AnalysisType,
	)

	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to trigger analysis", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, analysis, "analysis completed successfully")
}

// HandleGetAnalysis: GET /api/v1/attachments/{id}/analyses
func (h *ReportAIHandler) HandleGetAnalyses(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	attIDStr := r.PathValue("id")
	attID, err := uuid.Parse(attIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid attachment ID", "BAD_REQUEST", nil)
		return
	}

	analyses, err := h.svc.GetByAttachmentID(userCtx.TenantID, attID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list analyses", "INTERNAL_ERROR", err.Error())
		return
	}

	if analyses == nil {
		analyses = make([]ReportAIAnalysis, 0)
	}

	myhttp.RespondJSON(w, http.StatusOK, analyses, "analyses retrieved successfully")
}
