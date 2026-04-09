package routes

import (
	"net/http"

	myhttp "clinic-backend/internal/platform/http"
	"clinic-backend/internal/platform/http/router"
)

func registerPatientRoutes(mux *http.ServeMux, h Handlers) {
	api := router.NewGroup(mux, "/api/v1", myhttp.AuthMiddleware)

	patientRead := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist", "doctor"),
	)

	patientManage := api.Group(
		myhttp.RBACMiddleware("admin", "receptionist"),
	)

	visitManage := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	medicalRead := api.Group(
		myhttp.RBACMiddleware("admin", "doctor", "receptionist"),
	)

	medicalManage := api.Group(
		myhttp.RBACMiddleware("admin", "doctor"),
	)

	// Patients
	patientRead.Handle("GET", "/patients", http.HandlerFunc(h.PatientHandler.HandlePatients))
	patientManage.Handle("POST", "/patients", http.HandlerFunc(h.PatientHandler.HandlePatients))
	patientManage.Handle("GET", "/patients/{id}", http.HandlerFunc(h.PatientHandler.HandlePatientByID))
	patientManage.Handle("PUT", "/patients/{id}", http.HandlerFunc(h.PatientHandler.HandleUpdatePatient))
	patientManage.Handle("DELETE", "/patients/{id}", http.HandlerFunc(h.PatientHandler.HandleDeletePatient))

	// Patient profile
	patientRead.Handle("GET", "/patients/{id}/profile", http.HandlerFunc(h.PPHandler.GetProfile))
	patientRead.Handle("GET", "/patients/{id}/activities", http.HandlerFunc(h.PPHandler.GetActivityStream))
	patientRead.Handle("GET", "/patients/{id}/timeline", http.HandlerFunc(h.TimelineHandler.HandlePatientTimeline))

	// Visits
	visitManage.Handle("POST", "/visits", http.HandlerFunc(h.VisitHandler.HandleVisits))

	// Medical records
	medicalRead.Handle("GET", "/patients/{id}/medical-records", http.HandlerFunc(h.MedicalHandler.ListRecords))
	medicalManage.Handle("POST", "/patients/{id}/medical-records", http.HandlerFunc(h.MedicalHandler.CreateRecord))
	medicalRead.Handle("GET", "/medical-records/{id}", http.HandlerFunc(h.MedicalHandler.GetRecord))
	medicalManage.Handle("PATCH", "/medical-records/{id}", http.HandlerFunc(h.MedicalHandler.UpdateRecord))
	medicalManage.Handle("DELETE", "/medical-records/{id}", http.HandlerFunc(h.MedicalHandler.DeleteRecord))
	medicalManage.Handle("POST", "/medical-records/{id}/procedures", http.HandlerFunc(h.MedicalHandler.HandleAddProcedure))
}
