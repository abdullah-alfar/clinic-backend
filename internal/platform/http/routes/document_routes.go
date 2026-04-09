package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerDocumentRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	docRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	docManage := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	// Documents
	docRead.Handle("GET", "/patients/{id}/documents", http.HandlerFunc(h.DocumentHandler.HandleListPatientDocuments))
	docRead.Handle("POST", "/patients/{id}/documents", http.HandlerFunc(h.DocumentHandler.HandleUploadDocument))
	docManage.Handle("PATCH", "/documents/{id}", http.HandlerFunc(h.DocumentHandler.HandleUpdateDocument))
	docManage.Handle("DELETE", "/documents/{id}", http.HandlerFunc(h.DocumentHandler.HandleDeleteDocument))
	docRead.Handle("GET", "/documents/{id}/download", http.HandlerFunc(h.DocumentHandler.HandleDownloadDocument))
}
