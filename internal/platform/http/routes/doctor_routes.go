package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerDoctorRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	doctorRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	doctorAdmin := api.Group(
		myhttp.RBACMiddleware("admin"),
	)

	availRead := api.Group(
		myhttp.RBACMiddleware("admin", "doctor", "receptionist"),
	)

	availManage := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	// Doctors
	doctorRead.Handle("GET", "/doctors", http.HandlerFunc(h.DoctorHandler.ServeHTTP))
	doctorAdmin.Handle("POST", "/doctors", http.HandlerFunc(h.DoctorHandler.ServeHTTP))
	doctorAdmin.Handle("PUT", "/doctors/{id}", http.HandlerFunc(h.DoctorHandler.ServeHTTP))
	doctorAdmin.Handle("DELETE", "/doctors/{id}", http.HandlerFunc(h.DoctorHandler.ServeHTTP))

	// Advanced availability
	availRead.Handle("GET", "/doctors/{id}/availability", http.HandlerFunc(h.AvailabilityHandler.HandleGetFullAvailability))
	api.Handle("GET", "/doctors/{id}/availability/slots", http.HandlerFunc(h.AvailabilityHandler.HandleGetSlots))

	availManage.Handle("POST", "/doctors/{id}/availability/schedules", http.HandlerFunc(h.AvailabilityHandler.HandleCreateSchedule))
	availManage.Handle("PATCH", "/doctors/{id}/availability/schedules/{sid}", http.HandlerFunc(h.AvailabilityHandler.HandleUpdateSchedule))
	availManage.Handle("DELETE", "/doctors/{id}/availability/schedules/{sid}", http.HandlerFunc(h.AvailabilityHandler.HandleDeleteSchedule))
	availManage.Handle("POST", "/doctors/{id}/availability/schedules/{sid}/breaks", http.HandlerFunc(h.AvailabilityHandler.HandleCreateBreak))
	availManage.Handle("DELETE", "/doctors/{id}/availability/breaks/{bid}", http.HandlerFunc(h.AvailabilityHandler.HandleDeleteBreak))

	availRead.Handle("GET", "/doctors/{id}/availability/exceptions", http.HandlerFunc(h.AvailabilityHandler.HandleListExceptions))
	availManage.Handle("POST", "/doctors/{id}/availability/exceptions", http.HandlerFunc(h.AvailabilityHandler.HandleCreateException))
	availManage.Handle("DELETE", "/doctors/{id}/availability/exceptions/{eid}", http.HandlerFunc(h.AvailabilityHandler.HandleDeleteException))
}
