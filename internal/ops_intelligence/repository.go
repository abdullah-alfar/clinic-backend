package ops_intelligence

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	GetPatientAppointmentHistory(tenantID, patientID uuid.UUID) ([]AppointmentHistory, error)
	GetMedicalRecordsForAppointment(tenantID, apptID uuid.UUID) ([]MedicalRecordData, error)
	GetInvoicesForAppointment(tenantID, apptID uuid.UUID) ([]InvoiceData, error)
	GetCommunications(tenantID uuid.UUID, patientID *uuid.UUID, limit int) ([]Communication, error)
	GetPatientName(tenantID, patientID uuid.UUID) (string, error)
	CreateCommunication(c *Communication) error
}

type AppointmentHistory struct {
	ID        uuid.UUID
	Status    string
	StartTime time.Time
}

type MedicalRecordData struct {
	ID        uuid.UUID
	Notes     string
	Diagnosis string
}

type InvoiceData struct {
	ID     uuid.UUID
	Amount float64
	Status string
}

type postgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetPatientAppointmentHistory(tenantID, patientID uuid.UUID) ([]AppointmentHistory, error) {
	query := `
		SELECT id, status, start_time
		FROM appointments
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY start_time DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AppointmentHistory
	for rows.Next() {
		var h AppointmentHistory
		if err := rows.Scan(&h.ID, &h.Status, &h.StartTime); err != nil {
			return nil, err
		}
		results = append(results, h)
	}
	return results, nil
}

func (r *postgresRepository) GetMedicalRecordsForAppointment(tenantID, apptID uuid.UUID) ([]MedicalRecordData, error) {
	query := `
		SELECT id, notes, diagnosis
		FROM medical_records
		WHERE tenant_id = $1 AND appointment_id = $2
	`
	rows, err := r.db.Query(query, tenantID, apptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MedicalRecordData
	for rows.Next() {
		var m MedicalRecordData
		if err := rows.Scan(&m.ID, &m.Notes, &m.Diagnosis); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}

func (r *postgresRepository) GetInvoicesForAppointment(tenantID, apptID uuid.UUID) ([]InvoiceData, error) {
	query := `
		SELECT id, amount, status
		FROM invoices
		WHERE tenant_id = $1 AND appointment_id = $2
	`
	rows, err := r.db.Query(query, tenantID, apptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []InvoiceData
	for rows.Next() {
		var i InvoiceData
		if err := rows.Scan(&i.ID, &i.Amount, &i.Status); err != nil {
			return nil, err
		}
		results = append(results, i)
	}
	return results, nil
}

func (r *postgresRepository) GetCommunications(tenantID uuid.UUID, patientID *uuid.UUID, limit int) ([]Communication, error) {
	query := `
		SELECT id, tenant_id, patient_id, channel, direction, message, status, priority, category, created_at
		FROM communications
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if patientID != nil {
		query += " AND patient_id = $2"
		args = append(args, *patientID)
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Communication
	for rows.Next() {
		var c Communication
		if err := rows.Scan(&c.ID, &c.TenantID, &c.PatientID, &c.Channel, &c.Direction, &c.Message, &c.Status, &c.Priority, &c.Category, &c.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}

func (r *postgresRepository) GetPatientName(tenantID, patientID uuid.UUID) (string, error) {
	var firstName, lastName string
	err := r.db.QueryRow(`SELECT first_name, last_name FROM patients WHERE id = $1 AND tenant_id = $2`, patientID, tenantID).Scan(&firstName, &lastName)
	if err != nil {
		return "Unknown Patient", err
	}
	return firstName + " " + lastName, nil
}

func (r *postgresRepository) CreateCommunication(c *Communication) error {
	_, err := r.db.Exec(`
		INSERT INTO communications (id, tenant_id, patient_id, channel, direction, message, status, priority, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.TenantID, c.PatientID, c.Channel, c.Direction, c.Message, c.Status, c.Priority, c.Category)
	return err
}
