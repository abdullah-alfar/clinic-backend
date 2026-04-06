package rating

import "errors"

var (
	ErrAppointmentNotCompleted = errors.New("cannot rate an appointment that is not completed")
	ErrDuplicateRating         = errors.New("this appointment has already been rated")
	ErrUnauthorizedRating      = errors.New("you are not authorized to rate this appointment")
	ErrInvalidRatingValue      = errors.New("rating must be between 1 and 5")
	ErrRatingNotFound          = errors.New("rating not found")
)
