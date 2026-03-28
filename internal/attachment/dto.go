package attachment

type UploadAttachmentRequest struct {
	PatientID     string `form:"patient_id" validate:"required"`
	AppointmentID string `form:"appointment_id"`
}
