package auth

import "net/http"

func viewUserLogin(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Login Page"))
}

func viewUserLogout(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Logout Page"))
}

func viewUserRegister(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Register Page"))
}
