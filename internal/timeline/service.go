package timeline

import (
	"sort"

	"github.com/google/uuid"
)

type TimelineService struct {
	repo TimelineRepository
}

func NewTimelineService(repo TimelineRepository) *TimelineService {
	return &TimelineService{repo: repo}
}

func (s *TimelineService) GetPatientTimeline(tenantID, patientID uuid.UUID, filterType string, limit int) (*TimelineResponseDTO, error) {
	var allItems []TimelineItem

	// Decide which types to fetch based on filter
	fetchAppts := filterType == "" || filterType == string(TypeAppointment)
	fetchMedRecords := filterType == "" || filterType == string(TypeMedicalRecord)
	fetchInvoices := filterType == "" || filterType == string(TypeInvoice)
	fetchNotifications := filterType == "" || filterType == string(TypeNotification)
	fetchAttachments := filterType == "" || filterType == string(TypeAttachment)
	fetchNotes := filterType == "" || filterType == string(TypeNote)

	if fetchAppts {
		items, err := s.repo.GetPatientAppointments(tenantID, patientID)
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	if fetchMedRecords {
		items, err := s.repo.GetPatientMedicalRecords(tenantID, patientID)
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	if fetchInvoices {
		items, err := s.repo.GetPatientInvoices(tenantID, patientID)
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	if fetchNotifications {
		items, err := s.repo.GetPatientNotifications(tenantID, patientID)
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	if fetchAttachments {
		items, err := s.repo.GetPatientAttachments(tenantID, patientID)
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	if fetchNotes {
		items, err := s.repo.GetPatientVisits(tenantID, patientID)
		if err == nil {
			allItems = append(allItems, items...)
		}
	}

	// Sort by OccurredAt descending
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].OccurredAt.After(allItems[j].OccurredAt)
	})

	// Apply limit
	if limit > 0 && len(allItems) > limit {
		allItems = allItems[:limit]
	}

	// Map to DTO
	dtos := make([]TimelineItemDTO, len(allItems))
	for i, item := range allItems {
		dtos[i] = TimelineItemDTO{
			ID:          item.ID,
			Type:        item.Type,
			Title:       item.Title,
			Subtitle:    item.Subtitle,
			Description: item.Description,
			OccurredAt:  item.OccurredAt,
			Status:      item.Status,
			EntityID:    item.EntityID,
			EntityURL:   item.EntityURL,
			Metadata:    item.Metadata,
		}
	}

	return &TimelineResponseDTO{
		PatientID: patientID,
		Items:     dtos,
	}, nil
}
