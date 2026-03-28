package reportai

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"clinic-backend/internal/audit"
)

type ReportAIService struct {
	repo     Repository
	provider AIProvider
	audit    *audit.AuditService
}

func NewReportAIService(repo Repository, provider AIProvider, audit *audit.AuditService) *ReportAIService {
	return &ReportAIService{
		repo:     repo,
		provider: provider,
		audit:    audit,
	}
}

func (s *ReportAIService) RequestAnalysis(tenantID, patientID, attachmentID, userID uuid.UUID, fileURL, mimeType, analysisType string) (*ReportAIAnalysis, error) {
	// 1. Create a "pending" record
	analysis := &ReportAIAnalysis{
		ID:           uuid.New(),
		TenantID:     tenantID,
		PatientID:    patientID,
		AttachmentID: attachmentID,
		AnalysisType: analysisType,
		Status:       "pending",
		CreatedBy:    &userID,
	}

	if err := s.repo.Create(analysis); err != nil {
		return nil, err
	}

	// 2. Perform the analysis synchronously for now (could be moved to an async worker)
	summary, structData, err := s.provider.AnalyzeReport(context.Background(), fileURL, mimeType)

	if err != nil {
		analysis.Status = "failed"
		errMsg := err.Error()
		analysis.ErrorMessage = &errMsg
	} else {
		analysis.Status = "completed"
		analysis.Summary = &summary
		analysis.StructuredData = structData
	}

	// 3. Update the database record
	if err := s.repo.UpdateStatus(analysis); err != nil {
		fmt.Printf("failed to update analysis status: %v\n", err)
	}

	if s.audit != nil {
		s.audit.LogAction(tenantID, userID, "REPORT_AI_ANALYSIS", "attachment", attachmentID, map[string]string{
			"analysis_id": analysis.ID.String(),
			"status":      analysis.Status,
		})
	}

	return analysis, nil
}

func (s *ReportAIService) GetByAttachmentID(tenantID, attachmentID uuid.UUID) ([]ReportAIAnalysis, error) {
	return s.repo.GetByAttachmentID(tenantID, attachmentID)
}
