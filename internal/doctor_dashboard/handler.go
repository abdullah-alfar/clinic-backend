package doctor_dashboard

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	if userCtx.Role != "doctor" {
		myhttp.RespondError(w, http.StatusForbidden, "forbidden: doctor role required", "FORBIDDEN", nil)
		return
	}

	data, err := h.svc.GetDashboard(r.Context(), userCtx.TenantID, userCtx.UserID)
	if err != nil {
		if err == ErrDoctorNotFound {
			myhttp.RespondError(w, http.StatusNotFound, err.Error(), "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch dashboard", "DB_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, data, "success")
}
