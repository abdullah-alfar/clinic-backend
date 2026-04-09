package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerOpsAndSettingsRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	// Operational intelligence
	api.Handle("GET", "/appointments/{id}/no-show-risk", http.HandlerFunc(h.OpsHandler.HandleNoShowRisk))
	api.Handle("GET", "/revenue/missing", http.HandlerFunc(h.OpsHandler.HandleMissingRevenue))
	api.Handle("GET", "/communications", http.HandlerFunc(h.OpsHandler.HandleCommunications))

	// Smart scheduling
	api.Handle("GET", "/appointments/smart-suggestions", http.HandlerFunc(h.SchedulingHandler.HandleSmartSuggestions))

	// Settings
	api.Handle("GET", "/settings", http.HandlerFunc(h.SettingsHandler.HandleGetSettings))
	api.Handle("PUT", "/settings", http.HandlerFunc(h.SettingsHandler.HandleUpdateSettings))
	api.Handle("POST", "/settings/test-ai", http.HandlerFunc(h.SettingsHandler.HandleTestAI))
	api.Handle("POST", "/settings/test-email", http.HandlerFunc(h.SettingsHandler.HandleTestEmail))
	api.Handle("POST", "/settings/test-whatsapp", http.HandlerFunc(h.SettingsHandler.HandleTestWhatsApp))

	// AI Core
	api.Handle("POST", "/ai/chat", http.HandlerFunc(h.AIHandler.HandleChat))
}
