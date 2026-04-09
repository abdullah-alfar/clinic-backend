package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerPublicRoutes(mux *http.ServeMux, h Handlers) {
	// Public
	mux.HandleFunc("POST /api/v1/auth/login", h.AuthHandler.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.AuthHandler.HandleRefresh)
	mux.HandleFunc("GET /api/v1/tenants/config", h.TenantHandler.HandleGetConfig)

	// Webhooks
	mux.HandleFunc("POST /webhooks/whatsapp", h.BotHandler.HandleWebhook)

	// Authenticated base routes
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	api.Handle("GET", "/auth/me", http.HandlerFunc(h.AuthHandler.HandleMe))
	api.Handle("GET", "/search", http.HandlerFunc(h.SearchHandler.HandleSearch))
	api.Handle("GET", "/doctor-dashboard", http.HandlerFunc(h.DashHandler.GetDashboard))
}
