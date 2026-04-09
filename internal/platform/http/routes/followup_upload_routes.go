package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerFollowupUploadRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	uploadRoles := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	recurrenceRoles := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist"),
	)

	// Follow-ups
	api.Handle("POST", "/follow-ups", http.HandlerFunc(h.FollowupHandler.HandleCreate))
	api.Handle("GET", "/follow-ups", http.HandlerFunc(h.FollowupHandler.HandleList))
	api.Handle("PATCH", "/follow-ups/", http.HandlerFunc(h.FollowupHandler.HandleUpdateStatus))
	api.Handle("GET", "/patients/{id}/follow-ups", http.HandlerFunc(h.FollowupHandler.HandlePatientFollowUps))

	// Recurring appointments
	recurrenceRoles.Handle("POST", "/appointments/recurring", http.HandlerFunc(h.RecurrenceHandler.CreateRule))
	api.Handle("GET", "/appointments/recurring", http.HandlerFunc(h.RecurrenceHandler.ListRules))

	// Uploads
	uploadRoles.Handle("POST", "/uploads", http.HandlerFunc(h.UploadHandler.HandleUpload))
	api.Handle("GET", "/uploads/{tenant_id}/{file}", http.HandlerFunc(h.UploadHandler.HandleStatic))
}
