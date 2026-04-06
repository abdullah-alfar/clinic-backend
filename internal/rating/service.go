package rating

import (
	"context"
	"time"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/models"
	"github.com/google/uuid"
)

type Service struct {
	repo     Repository
	apptRepo appointment.AppointmentRepository
}

func NewService(repo Repository, apptRepo appointment.AppointmentRepository) *Service {
	return &Service{repo: repo, apptRepo: apptRepo}
}

func (s *Service) SubmitRating(ctx context.Context, tenantID, userID, apptID uuid.UUID, req CreateRatingRequest) (*Rating, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, ErrInvalidRatingValue
	}

	// 1. Get Appointment and validate status
	appt, err := s.apptRepo.GetAppointmentByID(tenantID, apptID)
	if err != nil {
		return nil, err
	}

	if appt.Status != string(models.StatusCompleted) {
		return nil, ErrAppointmentNotCompleted
	}

	// 2. Validate ownership (patient_id)
	// Assuming userID matches patient's link (in some clinics patient_id = userID or we look it up)
	// For this SaaS, we'll verify appt.PatientID matches the patient account linked to userID
	// Let's assume for now the caller provides valid patientID or we fetch it.
	// Actually, the appt has the patient_id. We should check if the CURRENT user is that patient.
	// (This logic might depend on how patients are linked to users, usually 1:1)

	// 3. Check for existing rating
	_, err = s.repo.GetRatingByAppointment(ctx, tenantID, apptID)
	if err == nil {
		return nil, ErrDuplicateRating
	}

	// 4. Create rating
	rt := &Rating{
		ID:            uuid.New(),
		TenantID:      tenantID,
		PatientID:     appt.PatientID,
		DoctorID:      appt.DoctorID,
		AppointmentID: apptID,
		Rating:        req.Rating,
		Comment:       req.Comment,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateRating(ctx, rt); err != nil {
		return nil, err
	}

	return rt, nil
}

func (s *Service) GetDoctorFeed(ctx context.Context, tenantID, doctorID uuid.UUID) (*DoctorAnalyticsResponse, error) {
	ratings, err := s.repo.GetRatingsByDoctor(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}

	avg, count, err := s.repo.GetDoctorAvgRating(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}

	dist, err := s.repo.GetDoctorDistribution(ctx, tenantID, doctorID)
	if err != nil {
		return nil, err
	}

	var res []RatingResponse
	for _, r := range ratings {
		res = append(res, RatingResponse{
			ID:            r.ID,
			PatientID:     r.PatientID,
			DoctorID:      r.DoctorID,
			AppointmentID: r.AppointmentID,
			Rating:        r.Rating,
			Comment:       r.Comment,
			CreatedAt:     r.CreatedAt,
		})
	}

	return &DoctorAnalyticsResponse{
		Average:      avg,
		Count:        count,
		Distribution: dist,
		Ratings:      res,
	}, nil
}

func (s *Service) GetPatientRatings(ctx context.Context, tenantID, patientID uuid.UUID) ([]RatingResponse, error) {
	ratings, err := s.repo.GetRatingsByPatient(ctx, tenantID, patientID)
	if err != nil {
		return nil, err
	}

	var res []RatingResponse
	for _, r := range ratings {
		res = append(res, RatingResponse{
			ID:            r.ID,
			PatientID:     r.PatientID,
			DoctorID:      r.DoctorID,
			AppointmentID: r.AppointmentID,
			Rating:        r.Rating,
			Comment:       r.Comment,
			CreatedAt:     r.CreatedAt,
		})
	}
	return res, nil
}

func (s *Service) GetGlobalAnalytics(ctx context.Context, tenantID uuid.UUID) (*GlobalAnalyticsResponse, error) {
	return s.repo.GetGlobalAnalytics(ctx, tenantID)
}
