package doctor_dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDashboard(ctx context.Context, tenantID, userID uuid.UUID) (*DashboardData, error) {
	// 1. Get Doctor Summary
	doctor, err := s.repo.GetDoctorByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, ErrDoctorNotFound
	}

	// 2. Handle Timezone for "Today"
	tzStr, err := s.repo.GetTenantTimezone(ctx, tenantID)
	if err != nil {
		tzStr = "UTC"
	}
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := todayStart.Add(24 * time.Hour)

	// 3. Aggregate Data
	stats, err := s.repo.GetStats(ctx, tenantID, doctor.ID, userID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}

	todayAppts, err := s.repo.GetTodayAppointments(ctx, tenantID, doctor.ID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}

	upcomingAppts, err := s.repo.GetUpcomingAppointments(ctx, tenantID, doctor.ID, todayEnd, 5)
	if err != nil {
		return nil, err
	}

	recentPatients, err := s.repo.GetRecentPatients(ctx, tenantID, doctor.ID, 5)
	if err != nil {
		return nil, err
	}

	medicalActivity, err := s.repo.GetRecentMedicalActivity(ctx, tenantID, doctor.ID, 5)
	if err != nil {
		return nil, err
	}

	return &DashboardData{
		Doctor:                 *doctor,
		Stats:                  *stats,
		TodayAppointments:      todayAppts,
		UpcomingAppointments:   upcomingAppts,
		RecentPatients:         recentPatients,
		RecentMedicalActivity: medicalActivity,
	}, nil
}
