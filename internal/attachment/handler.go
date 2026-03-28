package attachment

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type AttachmentHandler struct {
	svc *AttachmentService
}

func NewAttachmentHandler(svc *AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

// HandleUploadAttachment: POST /api/v1/patients/{id}/attachments
func (h *AttachmentHandler) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient ID", "BAD_REQUEST", nil)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); // 50MB max
	err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "file too large or invalid format", "BAD_REQUEST", err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "missing 'file' in request", "BAD_REQUEST", err.Error())
		return
	}
	defer file.Close()

	var apptID *uuid.UUID
	apptIDStr := r.FormValue("appointment_id")
	if apptIDStr != "" {
		parsed, err := uuid.Parse(apptIDStr)
		if err == nil {
			apptID = &parsed
		}
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	att, err := h.svc.UploadPatientFile(
		userCtx.TenantID,
		patientID,
		apptID,
		userCtx.UserID,
		header.Filename,
		mimeType,
		header.Size,
		file,
	)

	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to upload attachment", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, att, "attachment uploaded successfully")
}

// HandleListAttachments: GET /api/v1/patients/{id}/attachments
func (h *AttachmentHandler) HandleListAttachments(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient ID", "BAD_REQUEST", nil)
		return
	}

	atts, err := h.svc.GetPatientAttachments(userCtx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list attachments", "INTERNAL_ERROR", err.Error())
		return
	}

	// Make sure we never return null for a list
	if atts == nil {
		atts = make([]Attachment, 0)
	}

	myhttp.RespondJSON(w, http.StatusOK, atts, "attachments retrieved successfully")
}

// HandleGetAttachment: GET /api/v1/attachments/{id}
func (h *AttachmentHandler) HandleGetAttachment(w http.ResponseWriter, r *http.Request) {
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

	att, err := h.svc.GetAttachment(userCtx.TenantID, attID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			myhttp.RespondError(w, http.StatusNotFound, "attachment not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get attachment", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, att, "attachment retrieved successfully")
}

// HandleDeleteAttachment: DELETE /api/v1/attachments/{id}
func (h *AttachmentHandler) HandleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
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

	if err := h.svc.DeleteAttachment(userCtx.TenantID, attID, userCtx.UserID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			myhttp.RespondError(w, http.StatusNotFound, "attachment not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to delete attachment", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "attachment deleted successfully")
}
