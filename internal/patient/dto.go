package patient

type UpdatePatientRequest struct {
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"`
	Gender      *string `json:"gender"`
	Notes       *string `json:"notes"`
}
