package permissions

import (
	"net/http"
)

type PermissionTester interface {
	HasObjectPermission(r *http.Request, perm string, obj interface{}) bool
	HasPermission(r *http.Request, perm string) bool
}

var registry = &permissionTesterRegistry{
	testersByObj:  make(map[string][]PermissionTester),
	testersByPerm: make(map[string][]PermissionTester),
	objectHooks:   make([]func(r *http.Request, perm string, obj interface{}) bool, 0),
	stringHooks:   make([]func(r *http.Request, perm string) bool, 0),
}

// RegisterObject registers a permission tester for a specific object type.
func RegisterObject(obj interface{}, tester PermissionTester) {
	registry.registerForObj(obj, tester)
}

// RegisterPermission registers a permission tester for a specific permission.
func RegisterPermission(perm string, tester PermissionTester) {
	registry.registerForPerm(perm, tester)
}

// RegisterHook adds a hook to the permission tester registry.
func RegisterObjectHook(hook func(r *http.Request, perm string, obj interface{}) bool) {
	registry.registerObjectHook(hook)
}

func RegisterStringHook(hook func(r *http.Request, perm string) bool) {
	registry.registerStringHook(hook)
}

// HasObjectPermission checks if the given request has the permission to perform the given action on the given object.
func HasObjectPermission(r *http.Request, perm string, obj interface{}) bool {
	var t = registry.getTesterForObj(obj)
	return t.HasObjectPermission(r, perm, obj)
}

// HasPermission checks if the given request has the permission to perform the given action.
func HasPermission(r *http.Request, perm string) bool {
	var t = registry.getTesterForPerm(perm)
	return t.HasPermission(r, perm)
}

func Object(r *http.Request, perm string, obj interface{}) bool {
	return HasObjectPermission(r, perm, obj)
}

func String(r *http.Request, perm string) bool {
	return HasPermission(r, perm)
}
