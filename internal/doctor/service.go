package doctor

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"clinic-backend/internal/audit"
)

type Doctor struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"-"`
	UserID        *uuid.UUID `json:"user_id"`
	FullName      string     `json:"full_name"`
	Specialty     *string    `json:"specialty"`
	LicenseNumber *string    `json:"license_number"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type DoctorService struct {
	db    *sql.DB
	audit *audit.AuditService
}

func NewDoctorService(db *sql.DB, audit *audit.AuditService) *DoctorService {
	return &DoctorService{db: db, audit: audit}
}

func (s *DoctorService) Create(d *Doctor, actorID uuid.UUID) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()
	d.UpdatedAt = time.Now()

	_, err := s.db.Exec(`
		INSERT INTO doctors (id, tenant_id, user_id, full_name, specialty, license_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, d.ID, d.TenantID, d.UserID, d.FullName, d.Specialty, d.LicenseNumber, d.CreatedAt, d.UpdatedAt)

	if err == nil {
		s.audit.LogAction(d.TenantID, actorID, "CREATE_DOCTOR", "doctor", d.ID, d)
	}
	return err
}

func (s *DoctorService) List(tenantID uuid.UUID) ([]*Doctor, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, full_name, specialty, license_number, created_at, updated_at
		FROM doctors WHERE tenant_id = $1 ORDER BY full_name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Doctor
	for rows.Next() {
		var d Doctor
		if err := rows.Scan(&d.ID, &d.UserID, &d.FullName, &d.Specialty, &d.LicenseNumber, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &d)
	}
	return list, nil
}

func (s *DoctorService) Update(d *Doctor, actorID uuid.UUID) error {
	d.UpdatedAt = time.Now()
	res, err := s.db.Exec(`
		UPDATE doctors SET full_name = $1, specialty = $2, license_number = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`, d.FullName, d.Specialty, d.LicenseNumber, d.UpdatedAt, d.ID, d.TenantID)

	if err != nil {
		return err
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}

	s.audit.LogAction(d.TenantID, actorID, "UPDATE_DOCTOR", "doctor", d.ID, d)
	return nil
}

func (s *DoctorService) Delete(tenantID, id, actorID uuid.UUID) error {
	res, err := s.db.Exec(`DELETE FROM doctors WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}

	s.audit.LogAction(tenantID, actorID, "DELETE_DOCTOR", "doctor", id, nil)
	return nil
}
