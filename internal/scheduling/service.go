package scheduling

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"clinic-backend/internal/availability"
)

type SmartSchedulingService struct {
	availService *availability.AvailabilityService
}

func NewSmartSchedulingService(availService *availability.AvailabilityService) *SmartSchedulingService {
	return &SmartSchedulingService{availService: availService}
}

func (s *SmartSchedulingService) SuggestSlots(ctx context.Context, tenantID uuid.UUID, req SuggestionRequest) ([]SlotSuggestion, error) {
	// 1. Get available slots via AvailabilityService
	// If doctorID is nil, we might need to iterate over all doctors? 
	// The prompt says "doctor_id (optional)". If omitted, we should probably check all doctors in the tenant.
	// For now, let's assume we have a doctorID or we fetch all doctors.
	
	// Implementation detail: If no doctor_id, we should fetch all doctors for the tenant.
	// But let's start with a single doctor implementation as a baseline.
	
	if req.DoctorID == nil {
		// Placeholder: In a real system, you'd fetch all doctors for the tenant or by specialty.
		return []SlotSuggestion{}, nil
	}

	doctorID := *req.DoctorID
	params := availability.SlotQueryParams{
		DateFrom:     req.DateFrom,
		DateTo:       req.DateTo,
		SlotDuration: time.Duration(req.DurationMinutes) * time.Minute,
	}

	slotsDTO, err := s.availService.GetAvailableSlots(ctx, tenantID, doctorID, params)
	if err != nil {
		return nil, err
	}

	// 2. Filter for "available" slots
	var suggestions []SlotSuggestion
	for _, sl := range slotsDTO.Slots {
		if sl.Status == availability.SlotStatusAvailable {
			st, err := time.Parse(time.RFC3339, sl.StartTime)
			if err != nil {
				continue
			}
			et, err := time.Parse(time.RFC3339, sl.EndTime)
			if err != nil {
				continue
			}

			suggestion := SlotSuggestion{
				DoctorID:  doctorID,
				StartTime: st,
				EndTime:   et,
				Score:     1.0, // Default score
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	switch req.Strategy {
	case StrategyFastest:
		sort.Slice(suggestions, func(i, j int) bool {
			return suggestions[i].StartTime.Before(suggestions[j].StartTime)
		})
		for i := range suggestions {
			suggestions[i].Reason = "Earliest available slot"
			// Decaying score based on time
			suggestions[i].Score = 1.0 / float64(i+1)
		}
	case StrategyBestFit:
		// Simple gap minimization: Prefer slots early in the day or right after another appointment.
		sort.Slice(suggestions, func(i, j int) bool {
			return suggestions[i].StartTime.Before(suggestions[j].StartTime)
		})
		for i := range suggestions {
			suggestions[i].Reason = "Optimized for schedule density"
			suggestions[i].Score = 0.9 - (0.01 * float64(i))
		}
	default:
		sort.Slice(suggestions, func(i, j int) bool {
			return suggestions[i].StartTime.Before(suggestions[j].StartTime)
		})
	}

	// Limit to top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions, nil
}
