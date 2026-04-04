package whatsappbot

import (
	"encoding/json"
	"net/http"
	"time"

	myhttp "clinic-backend/internal/platform/http"

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

	// 1. Verify Token
	// Meta uses hub.verify_token for GET, but for POST we typically check a signature.
	// We'll use a simple query param or header for MVP.
	token := r.URL.Query().Get("token")
	if token != h.webhookToken {
		myhttp.RespondError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED", nil)
		return
	}

	// 2. Parse Payload
	var req WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid payload", "BAD_REQUEST", err.Error())
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		myhttp.RespondError(w, http.StatusBadRequest, "invalid tenant id", "BAD_REQUEST", nil)
		return
	}

	inbound := InboundMessage{
		From:          req.From,
		Body:          req.Body,
		ProviderMsgID: req.MsgID,
		ReceivedAt:    time.Now(),
	}

	// 3. Process
	// We use background context and return OK immediately to avoid provider timeouts.
	// For MVP, we'll do it synchronously for easier debugging.
	if err := h.svc.ProcessInbound(r.Context(), tenantID, inbound); err != nil {
		// Log error, but still return 200 OK so provider doesn't retry
		// fmt.Printf("Bot error: %v\n", err)
	}

	// Provider expects 200 OK
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
