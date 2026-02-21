package apperr

// ValidationError is returned when input data fails validation.
// The HTTP layer uses this type to distinguish 400 Bad Request from 500 Internal Server Error.
type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string {
	return e.msg
}

// NewValidation creates a ValidationError with the given message.
func NewValidation(msg string) *ValidationError {
	return &ValidationError{msg: msg}
}
