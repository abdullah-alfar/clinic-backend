package search

import (
	"context"
	"net/http"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type SearchHandler struct {
	svc SearchService
}

func NewSearchHandler(svc SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userCtx, ok := shared.GetUserContext(ctx)
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		myhttp.RespondJSON(w, http.StatusOK, SearchData{
			Patients: []PatientSearchResult{},
			Doctors:  []any{},
			Reports:  []any{},
		}, "success")
		return
	}

	data, searchErr := h.svc.GlobalSearch(ctx, userCtx.TenantID, query)
	if searchErr != nil {
		errStr := searchErr.Error()
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to execute search", "SEARCH_ERROR", errStr)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, data, "success")
}
