package rating

import (
	"time"

	"github.com/google/uuid"
)

type CreateRatingRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type RatingResponse struct {
	ID            uuid.UUID `json:"id"`
	PatientID     uuid.UUID `json:"patient_id"`
	PatientName   string    `json:"patient_name,omitempty"`
	DoctorID      uuid.UUID `json:"doctor_id"`
	AppointmentID uuid.UUID `json:"appointment_id"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

type DoctorAnalyticsResponse struct {
	Average      float64        `json:"average"`
	Count        int            `json:"count"`
	Distribution map[int]int    `json:"distribution"`
	Ratings      []RatingResponse `json:"ratings,omitempty"`
}

type GlobalAnalyticsResponse struct {
	TotalRatings      int               `json:"total_ratings"`
	AverageClinicRating float64           `json:"average_clinic_rating"`
	TopRatedDoctors    []DoctorRankEntry `json:"top_rated_doctors"`
	LowestRatedDoctors []DoctorRankEntry `json:"lowest_rated_doctors"`
}

type DoctorRankEntry struct {
	DoctorID   uuid.UUID `json:"doctor_id"`
	FullName   string    `json:"full_name"`
	Average    float64   `json:"average"`
	TotalCount int       `json:"total_count"`
}
