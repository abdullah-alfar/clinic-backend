package doctor_dashboard

type DoctorDashboardResponse struct {
	Data    *DashboardData `json:"data"`
	Message string         `json:"message"`
	Error   *string        `json:"error"`
}
