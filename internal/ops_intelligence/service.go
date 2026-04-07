package ops_intelligence

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	GetNoShowRisk(ctx context.Context, tenantID, apptID uuid.UUID, patientID uuid.UUID) (*NoShowRiskResponse, error)
	GetMissingRevenue(ctx context.Context, tenantID, apptID uuid.UUID) (*MissingRevenueResponse, error)
	GetCommunications(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID) ([]CommunicationResponse, error)
}

type opsService struct {
	repo      Repository
	predictor Predictor
	analyzer  Analyzer
	inbox     Inbox
}

func NewService(repo Repository, p Predictor, a Analyzer, i Inbox) Service {
	return &opsService{
		repo:      repo,
		predictor: p,
		analyzer:  a,
		inbox:     i,
	}
}

func (s *opsService) GetNoShowRisk(ctx context.Context, tenantID, apptID uuid.UUID, patientID uuid.UUID) (*NoShowRiskResponse, error) {
	history, err := s.repo.GetPatientAppointmentHistory(tenantID, patientID)
	if err != nil {
		return nil, err
	}

	risk := s.predictor.Predict(history)
	risk.AppointmentID = apptID

	return &NoShowRiskResponse{
		AppointmentID: risk.AppointmentID,
		RiskScore:     risk.RiskScore,
		RiskLevel:     risk.RiskLevel,
		Factors:       risk.Factors,
	}, nil
}

func (s *opsService) GetMissingRevenue(ctx context.Context, tenantID, apptID uuid.UUID) (*MissingRevenueResponse, error) {
	records, err := s.repo.GetMedicalRecordsForAppointment(tenantID, apptID)
	if err != nil {
		return nil, err
	}

	invoices, err := s.repo.GetInvoicesForAppointment(tenantID, apptID)
	if err != nil {
		return nil, err
	}

	missing := s.analyzer.Analyze(records, invoices)
	missing.AppointmentID = apptID

	return &MissingRevenueResponse{
		AppointmentID:   missing.AppointmentID,
		MissingServices: missing.MissingServices,
	}, nil
}

func (s *opsService) GetCommunications(ctx context.Context, tenantID uuid.UUID, patientID *uuid.UUID) ([]CommunicationResponse, error) {
	comms, err := s.repo.GetCommunications(tenantID, patientID, 50)
	if err != nil {
		return nil, err
	}

	var results []CommunicationResponse
	for _, c := range comms {
		patientName, _ := s.repo.GetPatientName(tenantID, c.PatientID)
		
		results = append(results, CommunicationResponse{
			ID:          c.ID,
			PatientID:   c.PatientID,
			PatientName: patientName,
			Channel:     c.Channel,
			Direction:   c.Direction,
			Message:     c.Message,
			Status:      c.Status,
			Priority:    c.Priority,
			Category:    c.Category,
			CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return results, nil
}
