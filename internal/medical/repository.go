package medical

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type MedicalRepository struct {
	db *sql.DB
}

func NewMedicalRepository(db *sql.DB) *MedicalRepository {
	return &MedicalRepository{db: db}
}

func (r *MedicalRepository) RunInTransaction(fn func(tx *sql.Tx) error) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	err = fn(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *MedicalRepository) CreateRecord(tx *sql.Tx, rec *MedicalRecord) error {
	rec.ID = uuid.New()
	rec.CreatedAt = time.Now()
	rec.UpdatedAt = time.Now()

	_, err := tx.Exec(`
		INSERT INTO medical_records (id, tenant_id, patient_id, appointment_id, doctor_id, diagnosis, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, rec.ID, rec.TenantID, rec.PatientID, rec.AppointmentID, rec.DoctorID, rec.Diagnosis, rec.Notes, rec.CreatedAt, rec.UpdatedAt)
	return err
}

func (r *MedicalRepository) CreateVital(tx *sql.Tx, v *MedicalVital) error {
	v.ID = uuid.New()
	v.CreatedAt = time.Now()

	_, err := tx.Exec(`
		INSERT INTO medical_vitals (id, medical_record_id, type, value, unit, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, v.ID, v.MedicalRecordID, v.Type, v.Value, v.Unit, v.CreatedAt)
	return err
}

func (r *MedicalRepository) CreateMedication(tx *sql.Tx, m *MedicalMedication) error {
	m.ID = uuid.New()
	m.CreatedAt = time.Now()

	_, err := tx.Exec(`
		INSERT INTO medical_medications (id, medical_record_id, name, dosage, frequency, duration, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, m.ID, m.MedicalRecordID, m.Name, m.Dosage, m.Frequency, m.Duration, m.Notes, m.CreatedAt)
	return err
}

func (r *MedicalRepository) GetRecordByID(tenantID, id uuid.UUID) (*MedicalRecord, error) {
	var rec MedicalRecord
	err := r.db.QueryRow(`
		SELECT id, tenant_id, patient_id, appointment_id, doctor_id, diagnosis, notes, created_at, updated_at
		FROM medical_records
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&rec.ID, &rec.TenantID, &rec.PatientID, &rec.AppointmentID, &rec.DoctorID,
		&rec.Diagnosis, &rec.Notes, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *MedicalRepository) GetVitalsByRecordID(recordID uuid.UUID) ([]*MedicalVital, error) {
	rows, err := r.db.Query(`
		SELECT id, medical_record_id, type, value, unit, created_at
		FROM medical_vitals
		WHERE medical_record_id = $1
		ORDER BY created_at ASC
	`, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vitals []*MedicalVital
	for rows.Next() {
		var v MedicalVital
		if err := rows.Scan(&v.ID, &v.MedicalRecordID, &v.Type, &v.Value, &v.Unit, &v.CreatedAt); err != nil {
			return nil, err
		}
		vitals = append(vitals, &v)
	}
	return vitals, nil
}

func (r *MedicalRepository) GetMedicationsByRecordID(recordID uuid.UUID) ([]*MedicalMedication, error) {
	rows, err := r.db.Query(`
		SELECT id, medical_record_id, name, dosage, frequency, duration, notes, created_at
		FROM medical_medications
		WHERE medical_record_id = $1
		ORDER BY created_at ASC
	`, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meds []*MedicalMedication
	for rows.Next() {
		var m MedicalMedication
		if err := rows.Scan(&m.ID, &m.MedicalRecordID, &m.Name, &m.Dosage, &m.Frequency, &m.Duration, &m.Notes, &m.CreatedAt); err != nil {
			return nil, err
		}
		meds = append(meds, &m)
	}
	return meds, nil
}

func (r *MedicalRepository) GetRecordsByPatientID(tenantID, patientID uuid.UUID) ([]*MedicalRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, tenant_id, patient_id, appointment_id, doctor_id, diagnosis, notes, created_at, updated_at
		FROM medical_records
		WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
	`, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*MedicalRecord
	for rows.Next() {
		var rec MedicalRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.PatientID, &rec.AppointmentID, &rec.DoctorID,
			&rec.Diagnosis, &rec.Notes, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, nil
}

func (r *MedicalRepository) UpdateRecord(tx *sql.Tx, tenantID uuid.UUID, rec *MedicalRecord) error {
	rec.UpdatedAt = time.Now()
	_, err := tx.Exec(`
		UPDATE medical_records
		SET diagnosis = $1, notes = $2, appointment_id = $3, updated_at = $4
		WHERE tenant_id = $5 AND id = $6
	`, rec.Diagnosis, rec.Notes, rec.AppointmentID, rec.UpdatedAt, tenantID, rec.ID)
	return err
}

func (r *MedicalRepository) DeleteVitalsByRecordID(tx *sql.Tx, recordID uuid.UUID) error {
	_, err := tx.Exec(`DELETE FROM medical_vitals WHERE medical_record_id = $1`, recordID)
	return err
}

func (r *MedicalRepository) DeleteMedicationsByRecordID(tx *sql.Tx, recordID uuid.UUID) error {
	_, err := tx.Exec(`DELETE FROM medical_medications WHERE medical_record_id = $1`, recordID)
	return err
}

func (r *MedicalRepository) DeleteRecord(tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM medical_records WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return err
}
