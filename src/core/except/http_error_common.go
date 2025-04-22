package except

func RaiseBadRequest(msg any, args ...any) ServerError {
	return NewServerError(400, msg, args...)
}

func RaiseUnauthorized(msg any, args ...any) ServerError {
	return NewServerError(401, msg, args...)
}

func RaiseForbidden(msg any, args ...any) ServerError {
	return NewServerError(403, msg, args...)
}

func RaiseNotFound(msg any, args ...any) ServerError {
	return NewServerError(404, msg, args...)
}

func RaiseMethodNotAllowed(msg any, args ...any) ServerError {
	return NewServerError(405, msg, args...)
}

func RaiseInternalServerError(msg any, args ...any) ServerError {
	return NewServerError(500, msg, args...)
}
