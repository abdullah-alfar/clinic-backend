package visit

import (
	"database/sql"
	"time"

	"clinic-backend/internal/audit"
	"clinic-backend/internal/models"
	"github.com/google/uuid"
)

type TimelineResponse struct {
	Appointments []*models.Appointment `json:"appointments"`
	Visits       []*models.Visit       `json:"visits"`
	Notes        []*models.Visit       `json:"notes"` // The prompt requested "notes" individually, we'll map visits to notes or just return visits.
}

type VisitService struct {
	db    *sql.DB
	audit *audit.AuditService
}

func NewVisitService(db *sql.DB, audit *audit.AuditService) *VisitService {
	return &VisitService{db: db, audit: audit}
}

func (s *VisitService) CreateVisit(v *models.Visit, actorID uuid.UUID) error {
	v.ID = uuid.New()
	v.CreatedAt = time.Now()

	_, err := s.db.Exec(`
		INSERT INTO visits (id, tenant_id, patient_id, appointment_id, doctor_id, notes, diagnosis, prescription, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, v.ID, v.TenantID, v.PatientID, v.AppointmentID, v.DoctorID, v.Notes, v.Diagnosis, v.Prescription, v.CreatedAt, v.CreatedAt)

	if err == nil {
		s.audit.LogAction(v.TenantID, actorID, "CREATE_VISIT", "visit", v.ID, v)
	}

	return err
}

func (s *VisitService) GetPatientTimeline(patientID string, tenantID uuid.UUID) (*TimelineResponse, error) {
	pID, err := uuid.Parse(patientID)
	if err != nil {
		return nil, err
	}

	appts, err := s.getAppointments(pID, tenantID)
	if err != nil {
		return nil, err
	}

	visits, err := s.getVisits(pID, tenantID)
	if err != nil {
		return nil, err
	}

	return &TimelineResponse{
		Appointments: appts,
		Visits:       visits,
		Notes:        visits, // Satisfying the explicit "notes" return requirement
	}, nil
}

func (s *VisitService) getAppointments(patientID, tenantID uuid.UUID) ([]*models.Appointment, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, patient_id, doctor_id, start_time, end_time, status, created_at
		FROM appointments
		WHERE patient_id = $1 AND tenant_id = $2
		ORDER BY start_time DESC
	`, patientID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appts []*models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(&a.ID, &a.TenantID, &a.PatientID, &a.DoctorID, &a.StartTime, &a.EndTime, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		appts = append(appts, &a)
	}
	return appts, nil
}

func (s *VisitService) getVisits(patientID, tenantID uuid.UUID) ([]*models.Visit, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, patient_id, appointment_id, doctor_id, coalesce(notes, ''), coalesce(diagnosis, ''), coalesce(prescription, ''), created_at
		FROM visits
		WHERE patient_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
	`, patientID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visits []*models.Visit
	for rows.Next() {
		var v models.Visit
		// appointment_id could be null in db. We need to handle nullable uuid. Let's use string or sql.NullString for scanning if needed,
		// but standard google/uuid implements sql.Scanner and handles NULL by becoming uuid.Nil.
		var apptID uuid.NullUUID
		if err := rows.Scan(&v.ID, &v.TenantID, &v.PatientID, &apptID, &v.DoctorID, &v.Notes, &v.Diagnosis, &v.Prescription, &v.CreatedAt); err != nil {
			return nil, err
		}
		v.AppointmentID = &apptID.UUID
		visits = append(visits, &v)
	}
	return visits, nil
}
