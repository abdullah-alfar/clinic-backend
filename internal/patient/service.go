package patient

import (
	"database/sql"
	"time"

	"clinic-backend/internal/audit"
	"errors"
	"github.com/google/uuid"
)

var ErrPatientNotFound = errors.New("patient not found")

type Patient struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"-"` // Hidden from response
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Phone       *string    `json:"phone"`
	Email       *string    `json:"email"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Gender      *string    `json:"gender"`
	Notes       *string    `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PatientService struct {
	db    *sql.DB
	audit *audit.AuditService
}

func NewPatientService(db *sql.DB, audit *audit.AuditService) *PatientService {
	return &PatientService{db: db, audit: audit}
}

func (s *PatientService) CreatePatient(p *Patient, actorID uuid.UUID) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()

	_, err := s.db.Exec(`
		INSERT INTO patients (id, tenant_id, first_name, last_name, phone, email, date_of_birth, gender, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, p.ID, p.TenantID, p.FirstName, p.LastName, p.Phone, p.Email, p.DateOfBirth, p.Gender, p.Notes, p.CreatedAt, p.CreatedAt)

	if err == nil {
		s.audit.LogAction(p.TenantID, actorID, "CREATE_PATIENT", "patient", p.ID, p)
	}

	return err
}

func (s *PatientService) ListPatients(tenantID uuid.UUID) ([]*Patient, error) {
	rows, err := s.db.Query(`
		SELECT id, first_name, last_name, phone, email, date_of_birth, gender, notes, created_at 
		FROM patients WHERE tenant_id = $1 ORDER BY last_name, first_name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patients []*Patient
	for rows.Next() {
		var p Patient
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Phone, &p.Email, &p.DateOfBirth, &p.Gender, &p.Notes, &p.CreatedAt); err != nil {
			return nil, err
		}
		patients = append(patients, &p)
	}
	return patients, nil
}

func (s *PatientService) GetPatientByID(id string, tenantID uuid.UUID) (*Patient, error) {
	patientID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	var p Patient
	err = s.db.QueryRow(`
		SELECT id, first_name, last_name, phone, email, date_of_birth, gender, notes, created_at
		FROM patients
		WHERE id = $1 AND tenant_id = $2
	`, patientID, tenantID).Scan(
		&p.ID,
		&p.FirstName,
		&p.LastName,
		&p.Phone,
		&p.Email,
		&p.DateOfBirth,
		&p.Gender,
		&p.Notes,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}

	return &p, nil
}
