package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerAppointmentRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	apptRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	apptCreate := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "patient"),
	)

	apptManageReception := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist"),
	)

	apptManageDoctor := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	apptManageAll := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	// Read
	api.Handle("GET", "/appointments/availability", http.HandlerFunc(h.AppointmentHandler.HandleGetAvailability))
	api.Handle("GET", "/appointments/next-available", http.HandlerFunc(h.AppointmentHandler.HandleGetNextAvailable))
	apptRead.Handle("GET", "/appointments/calendar", http.HandlerFunc(h.AppointmentHandler.HandleGetCalendar))
	apptRead.Handle("GET", "/appointments/{id}", http.HandlerFunc(h.AppointmentHandler.HandleGetByID))

	// Mutations
	apptCreate.Handle("POST", "/appointments", http.HandlerFunc(h.AppointmentHandler.HandleSchedule))
	apptManageReception.Handle("PATCH", "/appointments/{id}", http.HandlerFunc(h.AppointmentHandler.HandleUpdate))
	apptManageReception.Handle("PATCH", "/appointments/{id}/reschedule", http.HandlerFunc(h.AppointmentHandler.HandleReschedule))
	apptManageReception.Handle("PATCH", "/appointments/{id}/cancel", h.AppointmentHandler.HandleStatus("canceled"))
	apptManageDoctor.Handle("PATCH", "/appointments/{id}/confirm", h.AppointmentHandler.HandleStatus("confirmed"))
	apptManageDoctor.Handle("PATCH", "/appointments/{id}/complete", h.AppointmentHandler.HandleStatus("completed"))
	apptManageAll.Handle("PATCH", "/appointments/{id}/no-show", h.AppointmentHandler.HandleStatus("no_show"))
}
