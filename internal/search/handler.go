package search

import (
	"context"
	"net/http"
	"strings"
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
	typesParam := r.URL.Query().Get("types")

	var selectedTypes []string
	if strings.TrimSpace(typesParam) != "" {
		for _, t := range strings.Split(typesParam, ",") {
			selectedTypes = append(selectedTypes, strings.TrimSpace(t))
		}
	}

	data, searchErr := h.svc.GlobalSearch(ctx, userCtx.TenantID, query, selectedTypes)
	if searchErr != nil {
		errStr := searchErr.Error()
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to execute search", "SEARCH_ERROR", errStr)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, data, "success")
}
