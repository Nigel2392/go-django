package permissions

import "net/http"

type tester struct {
	hooks []func(r *http.Request, obj interface{}, perm string) bool
	tests []PermissionTester
}

func (t tester) HasObjectPermission(r *http.Request, obj interface{}, perm string) bool {
	if len(t.tests) == 0 && len(t.hooks) == 0 {
		return true
	}

	for _, hook := range t.hooks {
		if hook(r, obj, perm) {
			return true
		}
	}

	for _, tester := range t.tests {
		if tester.HasObjectPermission(r, obj, perm) {
			return true
		}
	}

	return false
}

func (t tester) HasPermission(r *http.Request, perm string) bool {
	if len(t.tests) == 0 && len(t.hooks) == 0 {
		return true
	}

	for _, hook := range t.hooks {
		if hook(r, nil, perm) {
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
