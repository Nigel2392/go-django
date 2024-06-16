package auth

import (
	"net/http"

	"github.com/a-h/templ"
)

var _ templ.Component

func viewUserLogin(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Login Page"))
}

func viewUserLogout(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Logout Page"))
}

func viewUserRegister(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Register Page"))
}
