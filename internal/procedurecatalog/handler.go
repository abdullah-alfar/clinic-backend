package procedurecatalog

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HandleListProcedures(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	procs, err := h.svc.ListProcedures(r.Context(), userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list procedures", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, procs, "procedures retrieved")
}

func (h *Handler) HandleGetProcedure(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	procIDStr := r.PathValue("id")
	procID, err := uuid.Parse(procIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid procedure ID", "BAD_REQUEST", err.Error())
		return
	}

	proc, err := h.svc.GetProcedure(r.Context(), userCtx.TenantID, procID)
	if err != nil {
		myhttp.RespondError(w, http.StatusNotFound, "procedure not found", "NOT_FOUND", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, proc, "procedure retrieved")
}

func (h *Handler) HandleCreateProcedure(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req CreateProcedureReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	proc, err := h.svc.CreateProcedure(r.Context(), userCtx.TenantID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create procedure", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, proc, "procedure created")
}

func (h *Handler) HandleUpdateProcedure(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	procIDStr := r.PathValue("id")
	procID, err := uuid.Parse(procIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid procedure ID", "BAD_REQUEST", err.Error())
		return
	}

	var req UpdateProcedureReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	proc, err := h.svc.UpdateProcedure(r.Context(), userCtx.TenantID, procID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update procedure", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, proc, "procedure updated")
}
