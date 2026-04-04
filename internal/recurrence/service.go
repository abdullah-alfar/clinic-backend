package recurrence

import (
	"context"
	"time"

	"clinic-backend/internal/appointment"
	"clinic-backend/internal/availability"

	"github.com/google/uuid"
)

type RecurrenceService struct {
	repo        RecurrenceRepository
	apptsRepo   appointment.AppointmentRepository
	availService *availability.AvailabilityService
}

func NewRecurrenceService(repo RecurrenceRepository, apptsRepo appointment.AppointmentRepository, availService *availability.AvailabilityService) *RecurrenceService {
	return &RecurrenceService{
		repo:        repo,
		apptsRepo:   apptsRepo,
		availService: availService,
	}
}

func (s *RecurrenceService) CreateRecurringAppointment(ctx context.Context, tenantID, actorID uuid.UUID, req CreateRecurrenceRequest) (*RecurrenceRule, []uuid.UUID, error) {
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	// Safe default: Limit to 1 year
	maxEndDate := time.Now().AddDate(1, 0, 0)
	if endDate.After(maxEndDate) {
		endDate = maxEndDate
	}

	rule := &RecurrenceRule{
		ID:         uuid.New(),
		TenantID:   tenantID,
		PatientID:  req.PatientID,
		DoctorID:   req.DoctorID,
		Frequency:  req.Frequency,
		Interval:   req.Interval,
		DayOfWeek:  req.DayOfWeek,
		DayOfMonth: req.DayOfMonth,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		StartDate:  startDate,
		EndDate:    endDate,
		Reason:     req.Reason,
		Status:     StatusActive,
	}

	if err := s.repo.CreateRule(ctx, rule); err != nil {
		return nil, nil, err
	}

	// Generate instances
	instances := s.generateInstances(rule)
	var createdApptIDs []uuid.UUID

	for _, inst := range instances {
		// Validate each instance against availability and conflicts
		err := s.availService.IsDoctorAvailableAt(ctx, tenantID, rule.DoctorID, inst.Start, inst.End)
		if err != nil {
			continue // Skip if doctor is not available
		}

		// Check for conflicts (existing appointments)
		count, err := s.apptsRepo.CheckConflictCount(tenantID, rule.DoctorID, inst.Start, inst.End, nil)
		if err != nil || count > 0 {
			continue // Skip if there is a conflict
		}

		// Create appointment
		appt := &appointment.Appointment{
			ID:               uuid.New(),
			TenantID:         tenantID,
			PatientID:        rule.PatientID,
			DoctorID:         rule.DoctorID,
			Status:           "scheduled",
			StartTime:        inst.Start,
			EndTime:          inst.End,
			Reason:           &rule.Reason,
			CreatedBy:        &actorID,
			RecurrenceRuleID: &rule.ID,
		}
		
		if err := s.apptsRepo.CreateAppointment(appt); err == nil {
			createdApptIDs = append(createdApptIDs, appt.ID)
		}
	}

	return rule, createdApptIDs, nil
}

type timeRange struct {
	Start time.Time
	End   time.Time
}

func (s *RecurrenceService) generateInstances(rule *RecurrenceRule) []timeRange {
	var instances []timeRange
	curr := rule.StartDate

	// Parse times
	st, _ := time.Parse("15:04:05", rule.StartTime)
	et, _ := time.Parse("15:04:05", rule.EndTime)

	for !curr.After(rule.EndDate) {
		match := false
		if rule.Frequency == FrequencyWeekly && rule.DayOfWeek != nil {
			if int(curr.Weekday()) == *rule.DayOfWeek {
				match = true
			}
		} else if rule.Frequency == FrequencyMonthly && rule.DayOfMonth != nil {
			if curr.Day() == *rule.DayOfMonth {
				match = true
			}
		}

		if match {
			instanceStart := time.Date(curr.Year(), curr.Month(), curr.Day(), st.Hour(), st.Minute(), st.Second(), 0, time.Local)
			instanceEnd := time.Date(curr.Year(), curr.Month(), curr.Day(), et.Hour(), et.Minute(), et.Second(), 0, time.Local)
			instances = append(instances, timeRange{instanceStart, instanceEnd})
		}
		
		curr = curr.AddDate(0, 0, 1)
	}
	return instances
}
