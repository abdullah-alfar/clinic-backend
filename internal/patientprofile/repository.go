package patientprofile

import (
	"database/sql"

	"clinic-backend/internal/patient"
	"github.com/google/uuid"
)

type PatientProfileRepository interface {
	GetPatient(tenantID, patientID uuid.UUID) (*patient.Patient, error)
	GetActivityStream(tenantID, patientID uuid.UUID, limit, offset int) ([]ActivityItemDTO, int, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) PatientProfileRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetPatient(tenantID, patientID uuid.UUID) (*patient.Patient, error) {
	var p patient.Patient
	err := r.db.QueryRow(`
		SELECT id, first_name, last_name, phone, email, date_of_birth, gender, created_at
		FROM patients
		WHERE id = $1 AND tenant_id = $2
	`, patientID, tenantID).Scan(
		&p.ID, &p.FirstName, &p.LastName, &p.Phone, &p.Email, &p.DateOfBirth, &p.Gender, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postgresRepository) GetActivityStream(tenantID, patientID uuid.UUID, limit, offset int) ([]ActivityItemDTO, int, error) {
	// 1. Get Total Count
	var total int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT id FROM appointments WHERE patient_id = $1 AND tenant_id = $2
			UNION ALL
			SELECT id FROM medical_records WHERE patient_id = $1 AND tenant_id = $2
			UNION ALL
			SELECT id FROM invoices WHERE patient_id = $1 AND tenant_id = $2
			UNION ALL
			SELECT id FROM whatsapp_messages WHERE patient_id = $1 AND tenant_id = $2 AND direction = 'outbound'
		) as combined
	`, patientID, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 2. Get Unified Stream
	rows, err := r.db.Query(`
		SELECT id, type, title, subtitle, status, occurred_at
		FROM (
			SELECT a.id, 'appointment' as type, 'Appointment' as title, COALESCE(d.full_name, 'Clinic Visit') as subtitle, a.status, a.start_time as occurred_at
			FROM appointments a
			LEFT JOIN doctors d ON d.id = a.doctor_id
			WHERE a.patient_id = $1 AND a.tenant_id = $2

			UNION ALL

			SELECT id, 'medical_record' as type, diagnosis as title, 'Clinical Note' as subtitle, 'completed' as status, created_at as occurred_at
			FROM medical_records
			WHERE patient_id = $1 AND tenant_id = $2

			UNION ALL

			SELECT id, 'invoice' as type, 'Invoice #' || LEFT(id::text, 8) as title, 'Amount: ' || status as subtitle, status, created_at as occurred_at
			FROM invoices
			WHERE patient_id = $1 AND tenant_id = $2

			UNION ALL

			SELECT id, 'communication' as type, 'Outbound Message' as title, LEFT(content, 100) as subtitle, 'sent' as status, created_at as occurred_at
			FROM whatsapp_messages
			WHERE patient_id = $1 AND tenant_id = $2 AND direction = 'outbound'
		) as stream
		ORDER BY occurred_at DESC
		LIMIT $3 OFFSET $4
	`, patientID, tenantID, limit, offset)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []ActivityItemDTO{}
	for rows.Next() {
		var item ActivityItemDTO
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Subtitle, &item.Status, &item.OccurredAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

