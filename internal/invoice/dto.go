package invoice

import "github.com/google/uuid"

type CreateInvoiceRequest struct {
	PatientID     uuid.UUID  `json:"patient_id"`
	AppointmentID *uuid.UUID `json:"appointment_id"`
	Amount        float64    `json:"amount"`
}

type InvoiceResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Error   interface{} `json:"error"`
}
