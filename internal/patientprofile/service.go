package patientprofile

import (
	"context"

	"github.com/google/uuid"
)

type PatientProfileService struct {
	repo PatientProfileRepository
}

func NewService(repo PatientProfileRepository) *PatientProfileService {
	return &PatientProfileService{repo: repo}
}

func (s *PatientProfileService) GetPatientProfile(ctx context.Context, tenantID, patientID uuid.UUID) (*PatientProfileData, error) {
	// 1. Get Patient Basic Info
	p, err := s.repo.GetPatient(tenantID, patientID)
	if err != nil {
		return nil, err
	}

	// 2. Flags Logic (Simplified for now since summary is gone, can be re-added via specific checks if needed)
	flags := []PatientFlag{}
	// Example: check for overdue invoices specifically if flags are critical
	// For now, returning empty flags or basic ones

	return &PatientProfileData{
		Patient: FromPatientModel(p),
		Flags:   flags,
	}, nil
}

func (s *PatientProfileService) GetActivityStream(ctx context.Context, tenantID, patientID uuid.UUID, page, limit int) (*ActivityStreamResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	items, total, err := s.repo.GetActivityStream(tenantID, patientID, limit, offset)
	if err != nil {
		return nil, err
	}

	return &ActivityStreamResponse{
		Data:       items,
		TotalItems: total,
		Page:       page,
		Limit:      limit,
		Message:    "Activity stream fetched successfully",
	}, nil
}

