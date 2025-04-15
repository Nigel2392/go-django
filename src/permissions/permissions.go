package permissions

import (
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type PermissionTester interface {
	HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool
	HasPermission(r *http.Request, perms ...string) bool
}

var Tester PermissionTester

func defaultLog(r *http.Request, perms ...string) bool {
	if LOG_IF_NONE_FOUND {
		logger.Warnf(
			"No permission testers found for \"%s\" (%s)",
			strings.Join(perms, "\", \""),
			r.URL.Path,
		)
	}

	var user = authentication.Retrieve(r)
	if user != nil {
		return user.IsAdmin() || DEFAULT_HAS_PERMISSION
	}

	return DEFAULT_HAS_PERMISSION
}

var (
	LOG_IF_NONE_FOUND      = true
	DEFAULT_IF_NONE_FOUND  = defaultLog
	DEFAULT_HAS_PERMISSION = false
)

// HasObjectPermission checks if the given request has the permission to perform the given action on the given object.
func HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	if len(perms) == 0 {
		return true
	}
	if Tester == nil {
		return DEFAULT_IF_NONE_FOUND(r, perms...)
	}
	return Tester.HasObjectPermission(r, obj, perms...)
}

// HasPermission checks if the given request has the permission to perform the given action.
func HasPermission(r *http.Request, perms ...string) bool {
	if len(perms) == 0 {
		return true
	}
	if Tester == nil {
		return DEFAULT_IF_NONE_FOUND(r, perms...)
	}
	return Tester.HasPermission(r, perms...)
}
