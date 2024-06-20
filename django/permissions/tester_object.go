package permissions

import (
	"net/http"

	"github.com/Nigel2392/django/core/logger"
)

var (
	LOG_IF_NONE_FOUND     = true
	DEFAULT_IF_NONE_FOUND = func(r *http.Request, perm string) bool {
		return true
	}
)

type tester struct {
	objectHooks []func(r *http.Request, perm string, obj interface{}) bool
	stringHooks []func(r *http.Request, perm string) bool
	tests       []PermissionTester
}

func (t *tester) HasObjectPermission(r *http.Request, perm string, obj interface{}) bool {
	if len(t.tests) == 0 && len(t.objectHooks) == 0 {
		if LOG_IF_NONE_FOUND {
			logger.Warnf("No permission testers found for type %T (%s)", obj, r.URL.Path)
		}
		return DEFAULT_IF_NONE_FOUND(r, perm)
	}

	for _, hook := range t.objectHooks {
		if hook(r, perm, obj) {
			return true
		}
	}

	for _, tester := range t.tests {
		if tester.HasObjectPermission(r, perm, obj) {
			return true
		}
	}

	return false
}

func (t *tester) HasPermission(r *http.Request, perm string) bool {
	if len(t.tests) == 0 && len(t.stringHooks) == 0 {
		if LOG_IF_NONE_FOUND {
			logger.Warnf("No permission testers found for %q (%s)", perm, r.URL.Path)
		}
		return DEFAULT_IF_NONE_FOUND(r, perm)
	}

	for _, hook := range t.stringHooks {
		if hook(r, perm) {
			return true
		}
	}
	for _, tester := range t.tests {
		if tester.HasPermission(r, perm) {
			return true
		}
	}
	return false
}
