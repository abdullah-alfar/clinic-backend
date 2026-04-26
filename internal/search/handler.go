package search

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

// SearchHandler wires HTTP to SearchService.
// It is responsible only for parsing/validating the request and
// delegating to the service — no business logic here.
type SearchHandler struct {
	svc SearchService
}

// NewSearchHandler constructs a SearchHandler.
func NewSearchHandler(svc SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// HandleSearch handles GET /api/v1/search
//
// Query parameters:
//
//	q          - search string (required, min 2 chars)
//	types      - comma-separated EntityType list (optional)
//	limit      - results per provider, 1-50 (optional, default 20)
//	date_from  - RFC3339 lower bound (optional)
//	date_to    - RFC3339 upper bound (optional)
//	status     - status filter string (optional)
//	patient_id - UUID filter (optional)
//	doctor_id  - UUID filter (optional)
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	userCtx, ok := shared.GetUserContext(ctx)
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < MinQueryLength {
		myhttp.RespondError(w, http.StatusBadRequest,
			"search query must be at least 2 characters", "QUERY_TOO_SHORT", nil)
		return
	}

	req := SearchRequest{
		TenantID: userCtx.TenantID,
		Query:    q,
		Limit:    DefaultLimitPerProvider,
	}

	// --- types ---
	if raw := strings.TrimSpace(r.URL.Query().Get("types")); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				req.Types = append(req.Types, t)
			}
		}
	}

	// --- limit ---
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			req.Limit = n
		}
	}

	// --- date_from ---
	if raw := r.URL.Query().Get("date_from"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			req.DateFrom = &t
		}
	}

	// --- date_to ---
	if raw := r.URL.Query().Get("date_to"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			req.DateTo = &t
		}
	}

	// --- status ---
	if raw := strings.TrimSpace(r.URL.Query().Get("status")); raw != "" {
		req.Status = raw
	}

	data, err := h.svc.GlobalSearch(ctx, req)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError,
			"failed to execute search", "SEARCH_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, data, "success")
}
