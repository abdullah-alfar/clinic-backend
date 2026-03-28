package invoice

import (
	"encoding/json"
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"
	"github.com/google/uuid"
)

type InvoiceHandler struct {
	svc *InvoiceService
}

func NewInvoiceHandler(svc *InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

func (h *InvoiceHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}
	tenantID := userCtx.TenantID

	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST", err.Error())
		return
	}

	inv, err := h.svc.CreateInvoice(req, tenantID)
	if err != nil {
		if err == ErrPatientNotFound || err == ErrApptNotFound {
			myhttp.RespondError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "Failed to create invoice", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusCreated, inv, "Invoice created successfully")
}

func (h *InvoiceHandler) HandleListPatientInvoices(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}
	tenantID := userCtx.TenantID

	patientIDStr := r.PathValue("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "Invalid patient ID", "INVALID_ID", nil)
		return
	}

	invoices, err := h.svc.ListPatientInvoices(patientID, tenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "Failed to list invoices", "INTERNAL_ERROR", nil)
		return
	}

	if invoices == nil {
		invoices = []*Invoice{}
	}

	myhttp.RespondJSON(w, http.StatusOK, invoices, "success")
}

func (h *InvoiceHandler) HandleMarkPaid(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}
	tenantID := userCtx.TenantID

	invoiceIDStr := r.PathValue("id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "Invalid invoice ID", "INVALID_ID", nil)
		return
	}

	err = h.svc.MarkAsPaid(invoiceID, tenantID)
	if err != nil {
		if err == ErrInvoiceNotFound {
			myhttp.RespondError(w, http.StatusNotFound, "Invoice not found", "NOT_FOUND", nil)
			return
		}
		myhttp.RespondError(w, http.StatusInternalServerError, "Failed to update invoice", "INTERNAL_ERROR", nil)
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, nil, "Invoice marked as paid")
}
