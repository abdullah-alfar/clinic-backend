package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerNotificationAttachmentRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	whatsappRead := api.Group(
		myhttp.RBACMiddleware("admin", "doctor", "receptionist"),
	)

	attachmentRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	attachmentManage := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	reportAIManage := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	// Notifications
	api.Handle("GET", "/notifications", http.HandlerFunc(h.NotificationHandler.HandleList))
	api.Handle("PATCH", "/notifications/{id}/read", http.HandlerFunc(h.NotificationHandler.HandleRead))
	api.Handle("GET", "/patients/{id}/notifications", http.HandlerFunc(h.NotificationHandler.HandlePatientHistory))
	api.Handle("GET", "/patients/{id}/notification-preferences", http.HandlerFunc(h.NotificationHandler.HandleGetPreferences))
	api.Handle("PUT", "/patients/{id}/notification-preferences", http.HandlerFunc(h.NotificationHandler.HandleUpdatePreferences))

	// WhatsApp bot
	whatsappRead.Handle("GET", "/patients/{id}/whatsapp/history", http.HandlerFunc(h.BotHandler.HandlePatientHistory))
	whatsappRead.Handle("GET", "/patients/{id}/whatsapp/status", http.HandlerFunc(h.BotHandler.HandleBotStatus))

	// Attachments
	attachmentRead.Handle("POST", "/patients/{id}/attachments", http.HandlerFunc(h.AttachmentHandler.HandleUploadAttachment))
	attachmentRead.Handle("GET", "/patients/{id}/attachments", http.HandlerFunc(h.AttachmentHandler.HandleListAttachments))
	attachmentRead.Handle("GET", "/attachments/{id}", http.HandlerFunc(h.AttachmentHandler.HandleGetAttachment))
	attachmentManage.Handle("DELETE", "/attachments/{id}", http.HandlerFunc(h.AttachmentHandler.HandleDeleteAttachment))

	// Report AI
	reportAIManage.Handle("POST", "/attachments/{id}/analyze", http.HandlerFunc(h.ReportAIHandler.HandleAnalyzeReport))
	reportAIManage.Handle("GET", "/attachments/{id}/analyses", http.HandlerFunc(h.ReportAIHandler.HandleGetAnalyses))
}
