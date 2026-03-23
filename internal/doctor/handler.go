package doctor

import (
	"encoding/json"
	"net/http"
	"strings"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type DoctorHandler struct {
	svc *DoctorService
}

func NewDoctorHandler(svc *DoctorService) *DoctorHandler {
	return &DoctorHandler{svc: svc}
}

func (h *DoctorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodPatch:
		h.handleUpdate(w, r)
	case http.MethodDelete:
		h.handleDelete(w, r)
	default:
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
	}
}

func (h *DoctorHandler) handleList(w http.ResponseWriter, r *http.Request) {
	userCtx, _ := shared.GetUserContext(r.Context())
	list, err := h.svc.List(userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		return
	}
	myhttp.RespondJSON(w, http.StatusOK, list, "success")
}

func (h *DoctorHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	userCtx, _ := shared.GetUserContext(r.Context())
	
	var d Doctor
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}
	
	d.TenantID = userCtx.TenantID
	if err := h.svc.Create(&d, userCtx.UserID); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create doctor", "CREATION_FAILED", nil)
		return
	}
	myhttp.RespondJSON(w, http.StatusCreated, d, "doctor created successfully")
}

func (h *DoctorHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	userCtx, _ := shared.GetUserContext(r.Context())
	
	// Extract ID from /api/v1/doctors/{id}
	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid doctor id", "BAD_REQUEST", nil)
		return
	}

	var d Doctor
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST", err.Error())
		return
	}

	d.ID = id
	d.TenantID = userCtx.TenantID

	if err := h.svc.Update(&d, userCtx.UserID); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update doctor", "UPDATE_FAILED", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, d, "doctor updated successfully")
}

func (h *DoctorHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	userCtx, _ := shared.GetUserContext(r.Context())
	
	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid doctor id", "BAD_REQUEST", nil)
		return
	}

	if err := h.svc.Delete(userCtx.TenantID, id, userCtx.UserID); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to delete doctor", "DELETE_FAILED", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "doctor deleted successfully")
}
