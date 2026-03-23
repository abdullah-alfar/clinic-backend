package tenant

import (
	"database/sql"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
)

type TenantHandler struct {
	svc *TenantService
}

func NewTenantHandler(svc *TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	subdomain := r.URL.Query().Get("subdomain")
	if subdomain == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "missing subdomain", "BAD_REQUEST", nil)
		return
	}

	theme, err := h.svc.GetTenantBySubdomain(subdomain)
	if err != nil {
		if err == sql.ErrNoRows {
			myhttp.RespondError(w, http.StatusNotFound, "tenant not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "internal error", "INTERNAL_ERROR", nil)
		return
	}

	// Format to match Phase 1 expected structure for UI
	res := map[string]interface{}{
		"id":       theme.ID,
		"name":     theme.Name,
		"logo_url": theme.LogoURL,
		"theme": map[string]string{
			"primaryColor":   getStringOrFallback(theme.PrimaryColor, "#0f172a"),
			"secondaryColor": getStringOrFallback(theme.SecondaryColor, "#64748b"),
			"borderRadius":   getStringOrFallback(theme.BorderRadius, "0.5rem"),
			"fontFamily":     getStringOrFallback(theme.FontFamily, "Inter"),
		},
	}

	myhttp.RespondJSON(w, http.StatusOK, res, "success")
}

func getStringOrFallback(val *string, fallback string) string {
	if val == nil || *val == "" {
		return fallback
	}
	return *val
}
