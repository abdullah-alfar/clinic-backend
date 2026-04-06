package rating

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Repository interface {
	CreateRating(ctx context.Context, r *Rating) error
	GetRatingByAppointment(ctx context.Context, tenantID, apptID uuid.UUID) (*Rating, error)
	GetRatingsByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]Rating, error)
	GetRatingsByPatient(ctx context.Context, tenantID, patientID uuid.UUID) ([]Rating, error)
	GetDoctorAvgRating(ctx context.Context, tenantID, doctorID uuid.UUID) (float64, int, error)
	GetDoctorDistribution(ctx context.Context, tenantID, doctorID uuid.UUID) (map[int]int, error)
	GetGlobalAnalytics(ctx context.Context, tenantID uuid.UUID) (*GlobalAnalyticsResponse, error)
}

type postgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateRating(ctx context.Context, rt *Rating) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ratings (id, tenant_id, patient_id, doctor_id, appointment_id, rating, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, rt.ID, rt.TenantID, rt.PatientID, rt.DoctorID, rt.AppointmentID, rt.Rating, rt.Comment, rt.CreatedAt)
	return err
}

func (r *postgresRepository) GetRatingByAppointment(ctx context.Context, tenantID, apptID uuid.UUID) (*Rating, error) {
	var rt Rating
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, patient_id, doctor_id, appointment_id, rating, coalesce(comment, ''), created_at
		FROM ratings WHERE tenant_id = $1 AND appointment_id = $2
	`, tenantID, apptID).Scan(&rt.ID, &rt.TenantID, &rt.PatientID, &rt.DoctorID, &rt.AppointmentID, &rt.Rating, &rt.Comment, &rt.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrRatingNotFound
	}
	return &rt, err
}

func (r *postgresRepository) GetRatingsByDoctor(ctx context.Context, tenantID, doctorID uuid.UUID) ([]Rating, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, patient_id, doctor_id, appointment_id, rating, coalesce(comment, ''), created_at
		FROM ratings WHERE tenant_id = $1 AND doctor_id = $2
		ORDER BY created_at DESC
	`, tenantID, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Rating
	for rows.Next() {
		var rt Rating
		if err := rows.Scan(&rt.ID, &rt.TenantID, &rt.PatientID, &rt.DoctorID, &rt.AppointmentID, &rt.Rating, &rt.Comment, &rt.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, rt)
	}
	return results, nil
}

func (r *postgresRepository) GetRatingsByPatient(ctx context.Context, tenantID, patientID uuid.UUID) ([]Rating, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, patient_id, doctor_id, appointment_id, rating, coalesce(comment, ''), created_at
		FROM ratings WHERE tenant_id = $1 AND patient_id = $2
		ORDER BY created_at DESC
	`, tenantID, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Rating
	for rows.Next() {
		var rt Rating
		if err := rows.Scan(&rt.ID, &rt.TenantID, &rt.PatientID, &rt.DoctorID, &rt.AppointmentID, &rt.Rating, &rt.Comment, &rt.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, rt)
	}
	return results, nil
}

func (r *postgresRepository) GetDoctorAvgRating(ctx context.Context, tenantID, doctorID uuid.UUID) (float64, int, error) {
	var avg float64
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT coalesce(avg(rating), 0), count(1)
		FROM ratings WHERE tenant_id = $1 AND doctor_id = $2
	`, tenantID, doctorID).Scan(&avg, &count)
	return avg, count, err
}

func (r *postgresRepository) GetDoctorDistribution(ctx context.Context, tenantID, doctorID uuid.UUID) (map[int]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT rating, count(1)
		FROM ratings WHERE tenant_id = $1 AND doctor_id = $2
		GROUP BY rating
	`, tenantID, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, err
		}
		res[rating] = count
	}
	return res, nil
}

func (r *postgresRepository) GetGlobalAnalytics(ctx context.Context, tenantID uuid.UUID) (*GlobalAnalyticsResponse, error) {
	var res GlobalAnalyticsResponse

	// Total & Avg
	err := r.db.QueryRowContext(ctx, `
		SELECT count(1), coalesce(avg(rating), 0)
		FROM ratings WHERE tenant_id = $1
	`, tenantID).Scan(&res.TotalRatings, &res.AverageClinicRating)
	if err != nil {
		return nil, err
	}

	// Rankings (Top 5)
	topRows, err := r.db.QueryContext(ctx, `
		SELECT r.doctor_id, d.full_name, avg(r.rating), count(r.id)
		FROM ratings r
		JOIN doctors d ON d.id = r.doctor_id
		WHERE r.tenant_id = $1
		GROUP BY r.doctor_id, d.full_name
		ORDER BY avg(r.rating) DESC, count(r.id) DESC
		LIMIT 5
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer topRows.Close()

	for topRows.Next() {
		var e DoctorRankEntry
		if err := topRows.Scan(&e.DoctorID, &e.FullName, &e.Average, &e.TotalCount); err != nil {
			return nil, err
		}
		res.TopRatedDoctors = append(res.TopRatedDoctors, e)
	}

	// Rankings (Bottom 5)
	bottomRows, err := r.db.QueryContext(ctx, `
		SELECT r.doctor_id, d.full_name, avg(r.rating), count(r.id)
		FROM ratings r
		JOIN doctors d ON d.id = r.doctor_id
		WHERE r.tenant_id = $1
		GROUP BY r.doctor_id, d.full_name
		ORDER BY avg(r.rating) ASC, count(r.id) DESC
		LIMIT 5
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer bottomRows.Close()

	for bottomRows.Next() {
		var e DoctorRankEntry
		if err := bottomRows.Scan(&e.DoctorID, &e.FullName, &e.Average, &e.TotalCount); err != nil {
			return nil, err
		}
		res.LowestRatedDoctors = append(res.LowestRatedDoctors, e)
	}

	return &res, nil
}
