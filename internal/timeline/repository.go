package timeline

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type TimelineRepository interface {
	GetPatientAppointments(tenantID, patientID uuid.UUID) ([]TimelineItem, error)
	GetPatientMedicalRecords(tenantID, patientID uuid.UUID) ([]TimelineItem, error)
	GetPatientInvoices(tenantID, patientID uuid.UUID) ([]TimelineItem, error)
	GetPatientNotifications(tenantID, patientID uuid.UUID) ([]TimelineItem, error)
	GetPatientAttachments(tenantID, patientID uuid.UUID) ([]TimelineItem, error)
	GetPatientVisits(tenantID, patientID uuid.UUID) ([]TimelineItem, error)
}

type postgresTimelineRepository struct {
	db *sql.DB
}

func NewPostgresTimelineRepository(db *sql.DB) TimelineRepository {
	return &postgresTimelineRepository{db: db}
}

func (r *postgresTimelineRepository) GetPatientAppointments(tenantID, patientID uuid.UUID) ([]TimelineItem, error) {
	query := `
		SELECT 
			a.id, a.tenant_id, a.patient_id, a.status, a.start_time, a.created_at,
			d.full_name as doctor_name, a.reason
		FROM appointments a
		LEFT JOIN doctors d ON d.id = a.doctor_id AND d.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.patient_id = $2
		ORDER BY a.start_time DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var id, tID, pID uuid.UUID
		var status string
		var startTime, createdAt time.Time
		var doctorName string
		var reason sql.NullString

		if err := rows.Scan(&id, &tID, &pID, &status, &startTime, &createdAt, &doctorName, &reason); err != nil {
			return nil, err
		}

		desc := "Appointment scheduled"
		if reason.Valid && reason.String != "" {
			desc = reason.String
		}

		items = append(items, TimelineItem{
			ID:          id,
			TenantID:    tID,
			PatientID:   pID,
			Type:        TypeAppointment,
			Title:       "Appointment " + status,
			Subtitle:    "Doctor: " + doctorName,
			Description: desc,
			OccurredAt:  startTime,
			Status:      &status,
			EntityID:    id,
			EntityURL:   "/appointments/" + id.String(),
			Metadata: map[string]any{
				"doctor_name": doctorName,
			},
		})
	}
	return items, nil
}

func (r *postgresTimelineRepository) GetPatientMedicalRecords(tenantID, patientID uuid.UUID) ([]TimelineItem, error) {
	query := `
		SELECT 
			mr.id, mr.tenant_id, mr.patient_id, mr.diagnosis, mr.created_at,
			d.full_name as doctor_name
		FROM medical_records mr
		LEFT JOIN doctors d ON d.id = mr.doctor_id AND d.tenant_id = mr.tenant_id
		WHERE mr.tenant_id = $1 AND mr.patient_id = $2
		ORDER BY mr.created_at DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var id, tID, pID uuid.UUID
		var diagnosis string
		var createdAt time.Time
		var doctorName string

		if err := rows.Scan(&id, &tID, &pID, &diagnosis, &createdAt, &doctorName); err != nil {
			return nil, err
		}

		items = append(items, TimelineItem{
			ID:          id,
			TenantID:    tID,
			PatientID:   pID,
			Type:        TypeMedicalRecord,
			Title:       "Medical Record Added",
			Subtitle:    "Diagnosis: " + diagnosis,
			Description: "Medical record and observations recorded by " + doctorName,
			OccurredAt:  createdAt,
			EntityID:    id,
			EntityURL:   "/medical-records/" + id.String(),
			Metadata: map[string]any{
				"doctor_name": doctorName,
				"diagnosis":   diagnosis,
			},
		})
	}
	return items, nil
}

func (r *postgresTimelineRepository) GetPatientInvoices(tenantID, patientID uuid.UUID) ([]TimelineItem, error) {
	query := `
		SELECT id, tenant_id, patient_id, amount, status, created_at
		FROM invoices
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var id, tID, pID uuid.UUID
		var amount float64
		var status string
		var createdAt time.Time

		if err := rows.Scan(&id, &tID, &pID, &amount, &status, &createdAt); err != nil {
			return nil, err
		}

		items = append(items, TimelineItem{
			ID:          id,
			TenantID:    tID,
			PatientID:   pID,
			Type:        TypeInvoice,
			Title:       "Invoice Created",
			Subtitle:    status,
			Description: "Invoice for services rendered.",
			OccurredAt:  createdAt,
			Status:      &status,
			EntityID:    id,
			EntityURL:   "/invoices/" + id.String(),
			Metadata: map[string]any{
				"amount": amount,
			},
		})
	}
	return items, nil
}

func (r *postgresTimelineRepository) GetPatientNotifications(tenantID, patientID uuid.UUID) ([]TimelineItem, error) {
	query := `
		SELECT id, tenant_id, patient_id, channel, subject, message, status, created_at
		FROM outbound_notifications
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var id, tID, pID uuid.UUID
		var channel, subject, message, status string
		var createdAt time.Time

		if err := rows.Scan(&id, &tID, &pID, &channel, &subject, &message, &status, &createdAt); err != nil {
			return nil, err
		}

		items = append(items, TimelineItem{
			ID:          id,
			TenantID:    tID,
			PatientID:   pID,
			Type:        TypeNotification,
			Title:       "Notification Sent",
			Subtitle:    channel + ": " + subject,
			Description: message,
			OccurredAt:  createdAt,
			Status:      &status,
			EntityID:    id,
			EntityURL:   "/notifications", 
			Metadata: map[string]any{
				"channel": channel,
				"subject": subject,
			},
		})
	}
	return items, nil
}

func (r *postgresTimelineRepository) GetPatientAttachments(tenantID, patientID uuid.UUID) ([]TimelineItem, error) {
	query := `
		SELECT id, tenant_id, patient_id, name, file_type, created_at
		FROM attachments
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var id, tID, pID uuid.UUID
		var name, fileType string
		var createdAt time.Time

		if err := rows.Scan(&id, &tID, &pID, &name, &fileType, &createdAt); err != nil {
			return nil, err
		}

		items = append(items, TimelineItem{
			ID:          id,
			TenantID:    tID,
			PatientID:   pID,
			Type:        TypeAttachment,
			Title:       "Document Uploaded",
			Subtitle:    name,
			Description: "File " + name + " (" + fileType + ") added to patient records.",
			OccurredAt:  createdAt,
			EntityID:    id,
			EntityURL:   "/attachments/" + id.String(),
			Metadata: map[string]any{
				"file_name": name,
				"file_type": fileType,
			},
		})
	}
	return items, nil
}

func (r *postgresTimelineRepository) GetPatientVisits(tenantID, patientID uuid.UUID) ([]TimelineItem, error) {
	query := `
		SELECT 
			v.id, v.tenant_id, v.patient_id, coalesce(v.notes, ''), v.created_at,
			d.full_name as doctor_name
		FROM visits v
		LEFT JOIN doctors d ON d.id = v.doctor_id AND d.tenant_id = v.tenant_id
		WHERE v.tenant_id = $1 AND v.patient_id = $2
		ORDER BY v.created_at DESC
	`
	rows, err := r.db.Query(query, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TimelineItem
	for rows.Next() {
		var id, tID, pID uuid.UUID
		var notes string
		var createdAt time.Time
		var doctorName string

		if err := rows.Scan(&id, &tID, &pID, &notes, &createdAt, &doctorName); err != nil {
			return nil, err
		}

		desc := notes
		if desc == "" {
			desc = "No additional notes provided."
		}

		items = append(items, TimelineItem{
			ID:          id,
			TenantID:    tID,
			PatientID:   pID,
			Type:        TypeNote,
			Title:       "Doctor Note Added",
			Subtitle:    "By " + doctorName,
			Description: desc,
			OccurredAt:  createdAt,
			EntityID:    id,
			EntityURL:   "/visits/" + id.String(),
			Metadata: map[string]any{
				"doctor_name": doctorName,
			},
		})
	}
	return items, nil
}
