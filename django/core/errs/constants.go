package errs

const (
	ErrFieldRequired Error = "Required field cannot be empty"
	ErrInvalidSyntax Error = "Invalid syntax for value"
	ErrInvalidType   Error = "Invalid type provided"
	ErrUnknown       Error = "Unknown error occurred"
)
