package search

import "github.com/google/uuid"

type SearchData struct {
	Patients []PatientSearchResult `json:"patients"`
	Doctors  []any                 `json:"doctors"` // Placeholder for future
	Reports  []any                 `json:"reports"` // Placeholder for future
}

type PatientSearchResult struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
}
