package document

import (
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
)

type DocumentHandler struct {
	svc *DocumentService
}

func NewDocumentHandler(svc *DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// HandleUploadDocument: POST /api/v1/patients/:id/documents
func (h *DocumentHandler) HandleUploadDocument(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(50 << 20); // 50MB
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

	category := DocumentCategory(r.FormValue("category"))
	if category == "" {
		category = CategoryGeneral
	}

	var apptID *uuid.UUID
	if val := r.FormValue("appointment_id"); val != "" {
		if id, err := uuid.Parse(val); err == nil {
			apptID = &id
		}
	}

	var medID *uuid.UUID
	if val := r.FormValue("medical_record_id"); val != "" {
		if id, err := uuid.Parse(val); err == nil {
			medID = &id
		}
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	doc, err := h.svc.UploadDocument(
		userCtx.TenantID,
		patientID,
		apptID,
		medID,
		userCtx.UserID,
		header.Filename,
		category,
		mimeType,
		header.Size,
		file,
	)

	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to upload document", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, ToDocumentResponse(doc), "document uploaded successfully")
}

// HandleListPatientDocuments: GET /api/v1/patients/:id/documents
func (h *DocumentHandler) HandleListPatientDocuments(w http.ResponseWriter, r *http.Request) {
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

	category := r.URL.Query().Get("category")

	docs, err := h.svc.GetPatientDocuments(userCtx.TenantID, patientID, category)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list documents", "INTERNAL_ERROR", err.Error())
		return
	}

	responses := make([]DocumentResponse, len(docs))
	for i, d := range docs {
		responses[i] = ToDocumentResponse(&d)
	}

	myhttp.RespondJSON(w, http.StatusOK, responses, "documents retrieved successfully")
}

// HandleUpdateDocument: PATCH /api/v1/documents/:id
func (h *DocumentHandler) HandleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	docIDStr := r.PathValue("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid document ID", "BAD_REQUEST", nil)
		return
	}

	var req UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	doc, err := h.svc.UpdateDocument(userCtx.TenantID, docID, userCtx.UserID, req)
	if err != nil {
		if err == ErrDocumentNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "document not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update document", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, ToDocumentResponse(doc), "document updated successfully")
}

// HandleDeleteDocument: DELETE /api/v1/documents/:id
func (h *DocumentHandler) HandleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	docIDStr := r.PathValue("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid document ID", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.DeleteDocument(userCtx.TenantID, docID, userCtx.UserID); err != nil {
		if err == ErrDocumentNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "document not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to delete document", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "document deleted successfully")
}

// HandleDownloadDocument: GET /api/v1/documents/:id/download
func (h *DocumentHandler) HandleDownloadDocument(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	docIDStr := r.PathValue("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid document ID", "BAD_REQUEST", nil)
		return
	}

	doc, err := h.svc.GetDocumentByID(userCtx.TenantID, docID)
	if err != nil {
		if err == ErrDocumentNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "document not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to get document", "INTERNAL_ERROR", err.Error())
		return
	}

	// Redirect to the storage path (if it's a URL) or serve the file
	// Since LocalFileStorage returns a /uploads/... URL, we can redirect or the frontend can just use the URL
	// But let's follow the requirement: GET /api/v1/documents/{id}/download
	http.Redirect(w, r, doc.StoragePath, http.StatusFound)
}
