package users

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type baseModelGetter interface {
	getBaseModel() *Base
}

type hasObjectPermissionsTester interface {
	HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool
}

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

	if tester, ok := user.(hasObjectPermissionsTester); ok {
		return tester.HasObjectPermission(r.Context(), obj, perms...)
	}

	if baseGetter, ok := user.(baseModelGetter); ok {
		var baseModel = baseGetter.getBaseModel()
		return baseModel.HasObjectPermission(r.Context(), obj, perms...)
	}

	assert.Fail(
		"User %T does not implement hasObjectPermissionsTester or baseModelGetter",
		user,
	)
	return false
}
