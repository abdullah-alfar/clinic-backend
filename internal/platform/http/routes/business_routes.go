package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerBusinessRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	reportRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	invoiceRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	invoiceManage := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist"),
	)

	inventoryRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	inventoryAdmin := api.Group(
		myhttp.RBACMiddleware("admin"),
	)

	inventoryAdjust := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist"),
	)

	procedureRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	procedureAdmin := api.Group(
		myhttp.RBACMiddleware("admin"),
	)

	ratingSubmit := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "patient"),
	)

	ratingPatientRead := api.Group(
		myhttp.RBACMiddleware("admin", "doctor", "receptionist"),
	)

	ratingAdmin := api.Group(
		myhttp.RBACMiddleware("admin"),
	)

	// Reporting
	reportRead.Handle("GET", "/dashboard/summary", http.HandlerFunc(h.ReportHandler.HandleDashboard))
	reportRead.Handle("GET", "/reports/appointments", http.HandlerFunc(h.ReportHandler.HandleAppointmentsReport))
	reportRead.Handle("GET", "/reports/patients", http.HandlerFunc(h.ReportHandler.HandlePatientsReport))

	// Invoices
	invoiceManage.Handle("POST", "/invoices", http.HandlerFunc(h.InvoiceHandler.HandleCreate))
	invoiceRead.Handle("GET", "/patients/{id}/invoices", http.HandlerFunc(h.InvoiceHandler.HandleListPatientInvoices))
	invoiceManage.Handle("PATCH", "/invoices/{id}/pay", http.HandlerFunc(h.InvoiceHandler.HandleMarkPaid))

	// Inventory
	inventoryRead.Handle("GET", "/inventory/items", http.HandlerFunc(h.InventoryHandler.HandleListItems))
	inventoryRead.Handle("GET", "/inventory/items/{id}", http.HandlerFunc(h.InventoryHandler.HandleGetItem))
	inventoryAdmin.Handle("POST", "/inventory/items", http.HandlerFunc(h.InventoryHandler.HandleCreateItem))
	inventoryAdmin.Handle("PATCH", "/inventory/items/{id}", http.HandlerFunc(h.InventoryHandler.HandleUpdateItem))
	inventoryAdjust.Handle("POST", "/inventory/items/{id}/adjust", http.HandlerFunc(h.InventoryHandler.HandleAdjustStock))
	inventoryAdmin.Handle("GET", "/inventory/items/{id}/movements", http.HandlerFunc(h.InventoryHandler.HandleListMovements))

	// Procedure catalog
	procedureRead.Handle("GET", "/procedures", http.HandlerFunc(h.ProcedureHandler.HandleListProcedures))
	procedureRead.Handle("GET", "/procedures/{id}", http.HandlerFunc(h.ProcedureHandler.HandleGetProcedure))
	procedureAdmin.Handle("POST", "/procedures", http.HandlerFunc(h.ProcedureHandler.HandleCreateProcedure))
	procedureAdmin.Handle("PATCH", "/procedures/{id}", http.HandlerFunc(h.ProcedureHandler.HandleUpdateProcedure))

	// Ratings
	ratingSubmit.Handle("POST", "/appointments/{id}/rating", http.HandlerFunc(h.RatingHandler.HandleSubmitRating))
	api.Handle("GET", "/doctors/{id}/ratings", http.HandlerFunc(h.RatingHandler.HandleGetDoctorRatings))
	ratingPatientRead.Handle("GET", "/patients/{id}/ratings", http.HandlerFunc(h.RatingHandler.HandleGetPatientRatings))
	ratingAdmin.Handle("GET", "/ratings/analytics", http.HandlerFunc(h.RatingHandler.HandleGetGlobalAnalytics))
}
