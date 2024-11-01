package errs

const (
	// Generic errors
	ErrFieldRequired Error = "Required field cannot be empty"
	ErrInvalidSyntax Error = "Invalid syntax for value"
	ErrInvalidType   Error = "Invalid type provided"
	ErrInvalidValue  Error = "Invalid value provided"
	ErrLengthMin     Error = "Minimum length not met"
	ErrLengthMax     Error = "Maximum length exceeded"
	ErrUnknown       Error = "Unknown error occurred"
)
