package inventory

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

func (h *Handler) HandleListItems(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	items, err := h.svc.ListItems(r.Context(), userCtx.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list items", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, items, "items retrieved")
}

func (h *Handler) HandleGetItem(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	itemIDStr := r.PathValue("id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid item ID", "BAD_REQUEST", err.Error())
		return
	}

	item, err := h.svc.GetItem(r.Context(), userCtx.TenantID, itemID)
	if err != nil {
		myhttp.RespondError(w, http.StatusNotFound, "item not found", "NOT_FOUND", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, item, "item retrieved")
}

func (h *Handler) HandleCreateItem(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	var req CreateItemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	item, err := h.svc.CreateItem(r.Context(), userCtx.TenantID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to create item", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, item, "item created")
}

func (h *Handler) HandleUpdateItem(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	itemIDStr := r.PathValue("id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid item ID", "BAD_REQUEST", err.Error())
		return
	}

	var req UpdateItemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	item, err := h.svc.UpdateItem(r.Context(), userCtx.TenantID, itemID, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to update item", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, item, "item updated")
}

func (h *Handler) HandleAdjustStock(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	itemIDStr := r.PathValue("id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid item ID", "BAD_REQUEST", err.Error())
		return
	}

	var uidPtr *uuid.UUID
	if userCtx.UserID != uuid.Nil {
		uidPtr = &userCtx.UserID
	}

	var req AdjustStockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	if err := h.svc.AdjustStock(r.Context(), userCtx.TenantID, itemID, uidPtr, req); err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to adjust stock", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "stock adjusted")
}

func (h *Handler) HandleListMovements(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	itemIDStr := r.PathValue("id")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid item ID", "BAD_REQUEST", err.Error())
		return
	}

	movements, err := h.svc.ListMovements(r.Context(), userCtx.TenantID, itemID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to list movements", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, movements, "movements retrieved")
}
