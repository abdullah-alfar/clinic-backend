package whatsappbot

import (
	"net/http"
	"time"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/shared"

	"github.com/google/uuid"
)

// WebhookPayload represents a simple generic webhook payload for dev purposes.
// In production, this would be the complex Meta/Twilio structure.
type WebhookPayload struct {
	TenantID string `json:"tenant_id"`
	From     string `json:"from"`
	Body     string `json:"body"`
	MsgID    string `json:"msg_id"`
}

type BotHandler struct {
	svc          *BotService
	webhookToken string
}

func NewBotHandler(svc *BotService, token string) *BotHandler {
	return &BotHandler{svc: svc, webhookToken: token}
}

// HandleWebhook receives messages from the WhatsApp Provider.
func (h *BotHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		myhttp.RespondError(w, http.StatusMethodNotAllowed, "method not allowed", "INVALID_METHOD", nil)
		return
	}

	// Simple MVP token check
	token := r.URL.Query().Get("token")
	if token != h.webhookToken {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// Twilio sends webhook payload as application/x-www-form-urlencoded, not JSON
	if err := r.ParseForm(); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	from := r.FormValue("From")
	body := r.FormValue("Body")
	msgID := r.FormValue("MessageSid")

	if from == "" {
		myhttp.RespondError(w, http.StatusBadRequest, "missing from", "BAD_REQUEST", nil)
		return
	}

	// TODO: replace this with proper tenant resolution logic
	// Example: derive tenant from Twilio "To" number or lookup from DB
	tenantID := uuid.MustParse("e830c33a-d04b-4888-91ed-846114eb16eb")

	inbound := InboundMessage{
		From:          from,
		Body:          body,
		ProviderMsgID: msgID,
		ReceivedAt:    time.Now(),
	}

	// Process synchronously for easier debugging in MVP
	if err := h.svc.ProcessInbound(r.Context(), tenantID, inbound); err != nil {
		// Log error if you have logger, but still return 200 so Twilio doesn't retry endlessly
		// fmt.Printf("Bot error: %v\n", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *BotHandler) HandlePatientHistory(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	history, err := h.svc.GetPatientHistory(r.Context(), uctx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch history", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, history, "History fetched successfully")
}

func (h *BotHandler) HandleBotStatus(w http.ResponseWriter, r *http.Request) {
	patientID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid patient id", "BAD_REQUEST", nil)
		return
	}

	uctx, ok := shared.GetUserContext(r.Context())
	if !ok {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	status, err := h.svc.GetBotStatus(r.Context(), uctx.TenantID, patientID)
	if err != nil {
		myhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch status", "INTERNAL_ERROR", err.Error())
		return
	}

	myhttp.RespondJSON(w, http.StatusOK, status, "Status fetched successfully")
}
