package doctor_dashboard

import "errors"

var (
	ErrDoctorNotFound    = errors.New("doctor profile not found for this user")
	ErrUnauthorizedRole = errors.New("unauthorized: current user does not have doctor role")
	ErrFailedToFetchData = errors.New("failed to aggregate dashboard data")
)
