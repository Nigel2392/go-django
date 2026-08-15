package errs

const (
	// Generic errors
	ErrNotImplemented Error = "This operation is not implemented"
	ErrFieldRequired  Error = "Required field cannot be empty"
	ErrInvalidSyntax  Error = "Invalid syntax for value"
	ErrInvalidType    Error = "Invalid type provided"
	ErrInvalidValue   Error = "Invalid value provided"
	ErrLengthMin      Error = "Minimum length not met"
	ErrLengthMax      Error = "Maximum length exceeded"
	ErrUnknown        Error = "Unknown error occurred"
)

//
//	const (
//		// Generic errors
//		errNotImplemented Error = "This operation is not implemented"
//		errFieldRequired  Error = "Required field cannot be empty"
//		errInvalidSyntax  Error = "Invalid syntax for value"
//		errInvalidType    Error = "Invalid type provided"
//		errInvalidValue   Error = "Invalid value provided"
//		errLengthMin      Error = "Minimum length not met"
//		errLengthMax      Error = "Maximum length exceeded"
//		ErrUnknown        Error = "Unknown error occurred"
//	)
//
//	var (
//		ErrNotImplemented = errors.NotImplemented.WithCause(errNotImplemented)
//		ErrFieldRequired  = errors.FieldNull.WithCause(errFieldRequired)
//		ErrInvalidSyntax  = errors.SyntaxError.WithCause(errInvalidSyntax)
//		ErrInvalidType    = errors.TypeMismatch.WithCause(errInvalidType)
//		ErrInvalidValue   = errors.ValueError.WithCause(errInvalidValue)
//		ErrLengthMin      = errors.ValueError.Wrap("Minimum length not met").WithCause(errLengthMin)
//		ErrLengthMax      = errors.ValueError.Wrap("Maximum length exceeded").WithCause(errLengthMax)
//	)
//
//
