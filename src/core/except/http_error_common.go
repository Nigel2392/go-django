package except

import "net/http"

func RaiseBadRequest(msg any, args ...any) ServerError {
	return NewServerError(http.StatusBadRequest, msg, args...)
}

func RaiseUnauthorized(msg any, args ...any) ServerError {
	return NewServerError(http.StatusUnauthorized, msg, args...)
}

func RaiseForbidden(msg any, args ...any) ServerError {
	return NewServerError(http.StatusForbidden, msg, args...)
}

func RaiseNotFound(msg any, args ...any) ServerError {
	return NewServerError(http.StatusNotFound, msg, args...)
}

func RaiseMethodNotAllowed(msg any, args ...any) ServerError {
	return NewServerError(http.StatusMethodNotAllowed, msg, args...)
}

func RaiseInternalServerError(msg any, args ...any) ServerError {
	return NewServerError(http.StatusInternalServerError, msg, args...)
}
