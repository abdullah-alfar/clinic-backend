package upload

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"fmt"
	"strings"

	"github.com/google/uuid"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"clinic-backend/internal/audit"
)

type UploadHandler struct {
	db    *sql.DB
	audit *audit.AuditService
}

func NewUploadHandler(db *sql.DB, audit *audit.AuditService) *UploadHandler {
	return &UploadHandler{db: db, audit: audit}
}

func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// Parse multipart form (10 MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "file too large or invalid form", "BAD_REQUEST", err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "missing 'file' in form", "BAD_REQUEST", err.Error())
		return
	}
	defer file.Close()

	patientIDStr := r.FormValue("patient_id")
	if patientIDStr == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "missing 'patient_id' in form", "BAD_REQUEST", nil)
		return
	}

	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient_id", "BAD_REQUEST", nil)
		return
	}

	// Setup tenant directory structure
	tenantDir := filepath.Join(".", "uploads", userCtx.TenantID.String())
	if err := os.MkdirAll(tenantDir, os.ModePerm); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create directory", "INTERNAL_ERROR", nil)
		return
	}

	// Create unique file name
	fileID := uuid.New()
	ext := filepath.Ext(header.Filename)
	uniqueFileName := fmt.Sprintf("%s%s", fileID.String(), ext)
	dstPath := filepath.Join(tenantDir, uniqueFileName)

	dst, err := os.Create(dstPath)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to save file on server", "INTERNAL_ERROR", err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to write file body", "INTERNAL_ERROR", err.Error())
		return
	}

	// Generate relative string for db/client
	// example: /uploads/tenant_id/uuid.ext
	fileURL := fmt.Sprintf("/uploads/%s/%s", userCtx.TenantID.String(), uniqueFileName)
	fileType := header.Header.Get("Content-Type")

	// Store in DB attachments table
	_, err = h.db.Exec(`
		INSERT INTO attachments (id, tenant_id, patient_id, file_url, file_type, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, fileID, userCtx.TenantID, patientID, fileURL, fileType, userCtx.UserID)

	if err != nil {
		// Cleanup physical file since DB failed
		os.Remove(dstPath)
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to execute database insertion", "INTERNAL_ERROR", err.Error())
		return
	}

	h.audit.LogAction(userCtx.TenantID, userCtx.UserID, "UPLOAD_ATTACHMENT", "attachment", fileID, map[string]string{"file_url": fileURL})

	myhttp.RespondJSON(w, http.StatusCreated, map[string]string{
		"id":       fileID.String(),
		"file_url": fileURL,
	}, "file uploaded successfully")
}

func (h *UploadHandler) HandleStatic(w http.ResponseWriter, r *http.Request) {
	// Simple static file router locking directly to path segments
	// Endpoint: /uploads/{tenant_id}/{file}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		myhttp.RespondError(w, http.StatusForbidden, "invalid access", "FORBIDDEN", nil)
		return
	}

	tenantID := parts[2]
	fileName := parts[3]

	// Security: Prevent directory traversal
	if strings.Contains(fileName, "..") || strings.Contains(tenantID, "..") {
		myhttp.RespondError(w, http.StatusForbidden, "invalid path traversal", "FORBIDDEN", nil)
		return
	}

	filePath := filepath.Join(".", "uploads", tenantID, fileName)
	http.ServeFile(w, r, filePath)
}
