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

	// 2. Get Summary Snapshot
	summary, err := s.repo.GetSummary(tenantID, patientID)
	if err != nil {
		return nil, err
	}

	// 3. Flags Logic
	flags := []PatientFlag{}
	if summary.NoShowCount > 0 {
		flags = append(flags, PatientFlag{Type: "alert", Label: "No-show history detected"})
	}
	if summary.UnpaidInvoicesCount > 0 {
		flags = append(flags, PatientFlag{Type: "billing", Label: "Outstanding balance"})
	}
	if summary.TotalAppointments > 0 && summary.LastVisitAt == nil && summary.UpcomingAppointmentAt == nil {
		flags = append(flags, PatientFlag{Type: "info", Label: "Inactive patient"})
	}

	// 4. Recent Activity
	limit := 5
	appointments, _ := s.repo.GetRecentAppointments(tenantID, patientID, limit)
	medicalRecords, _ := s.repo.GetRecentMedicalRecords(tenantID, patientID, limit)
	reports, _ := s.repo.GetRecentReports(tenantID, patientID, limit)
	invoices, _ := s.repo.GetRecentInvoices(tenantID, patientID, limit)

	return &PatientProfileData{
		Patient: FromPatientModel(p),
		Summary: summary,
		Flags:   flags,
		RecentActivity: PatientRecentActivity{
			Appointments:   appointments,
			MedicalRecords: medicalRecords,
			Invoices:       invoices,
			Reports:        reports,
			Communications: []RecentActivity{}, // Placeholder
		},
	}, nil
}
