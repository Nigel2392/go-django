package auth

import (
	"net/http"

	"github.com/Nigel2392/mux/middleware/authentication"
)

type PermissionsBackend struct{}

func NewPermissionsBackend() *PermissionsBackend {
	return &PermissionsBackend{}
}

func (pb *PermissionsBackend) HasPermission(r *http.Request, perms ...string) bool {
	return pb.HasObjectPermission(r, nil, perms...)
}

func (pb *PermissionsBackend) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	var user = authentication.Retrieve(r)
	if user == nil {
		return false
	}

	if len(perms) == 0 {
		return true
	}

	if !user.IsAuthenticated() {
		return false
	}

	if user.IsAdmin() {
		return true
	}

	var u = user.(*User)
	return u.HasObjectPermission(obj, perms...)
}
