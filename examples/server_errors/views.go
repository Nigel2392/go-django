package main

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux"
)

func Index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the Index page!\n"))
}

func RaiseErrorCode(w http.ResponseWriter, r *http.Request) {
	var vars = mux.Vars(r)
	var code = vars.GetInt("code")
	if code == 0 {
		code = http.StatusInternalServerError
	}

	except.Fail(code, "This is a test error: %d", code)
}

func Handle500(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("An internal server error occurred, please try again later.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}

func Handle404(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("The page you are looking for does not exist.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}

func Handle403(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("You do not have permission to access this resource.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}

func Handle400(w http.ResponseWriter, r *http.Request, err except.ServerError) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(http.StatusText(err.StatusCode()) + ": "))
	w.Write([]byte("The request was invalid or cannot be served.\n"))
	w.Write([]byte(err.UserMessage() + "\n"))
}
