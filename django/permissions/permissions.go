package permissions

import (
	"net/http"
)

type PermissionTester interface {
	HasObjectPermission(r *http.Request, obj interface{}, perm string) bool
	HasPermission(r *http.Request, perm string) bool
}

var registry = &permissionTesterRegistry{}

// RegisterObject registers a permission tester for a specific object type.
func RegisterObject(obj interface{}, tester PermissionTester) {
	registry.registerForObj(obj, tester)
}

// RegisterPermission registers a permission tester for a specific permission.
func RegisterPermission(perm string, tester PermissionTester) {
	registry.registerForPerm(perm, tester)
}

// RegisterHook adds a hook to the permission tester registry.
func RegisterHook(hook func(r *http.Request, obj interface{}, perm string) bool) {
	registry.registerHook(hook)
}

// HasObjectPermission checks if the given request has the permission to perform the given action on the given object.
func HasObjectPermission(r *http.Request, obj interface{}, perm string) bool {
	var t = registry.getTesterForObj(obj)
	return t.HasObjectPermission(r, obj, perm)
}

// HasPermission checks if the given request has the permission to perform the given action.
func HasPermission(r *http.Request, perm string) bool {
	var t = registry.getTesterForPerm(perm)
	return t.HasPermission(r, perm)
}

func Object(r *http.Request, obj interface{}, perm string) bool {
	return HasObjectPermission(r, obj, perm)
}

func String(r *http.Request, perm string) bool {
	return HasPermission(r, perm)
}
