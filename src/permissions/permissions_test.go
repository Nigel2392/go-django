package permissions_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nigel2392/go-django/src/permissions"
)

// NewContextWithPermissions creates a new context with the given permissions.
func NewContextWithPermissions(ctx context.Context, perms ...string) context.Context {
	var permissionMap = make(map[string]struct{}, len(perms))
	for _, perm := range perms {
		permissionMap[perm] = struct{}{}
	}
	return context.WithValue(ctx, "permissions", permissionMap)
}

func GetRequestWithPermissions(perms ...string) *http.Request {
	var request = httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(
		NewContextWithPermissions(request.Context(), perms...),
	)
	return request
}

var _ permissions.PermissionTester = &TestPermissionTester{}

type TestPermissionTester struct {
	HasObjectPermissionFunc func(r *http.Request, obj interface{}, perms ...string) bool
	HasPermissionFunc       func(r *http.Request, perms ...string) bool
}

func (t *TestPermissionTester) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	if t.HasObjectPermissionFunc != nil {
		return t.HasObjectPermissionFunc(r, obj, perms...)
	}
	return false
}

func (t *TestPermissionTester) HasPermission(r *http.Request, perms ...string) bool {
	if t.HasPermissionFunc != nil {
		return t.HasPermissionFunc(r, perms...)
	}
	return false
}

func TestNewContextWithPermissions(t *testing.T) {
	var ctx = NewContextWithPermissions(context.Background(), "can_view")
	if ctx == nil {
		t.Fatal("expected context, got nil")
	}
	if _, ok := ctx.Value("permissions").(map[string]struct{}); !ok {
		t.Fatalf("expected map[string]struct{}, got %T", ctx.Value("permissions"))
	}
}

func TestHasPermissions(t *testing.T) {
	var tester = &TestPermissionTester{
		HasPermissionFunc: func(r *http.Request, perms ...string) bool {
			var permissionMap = r.Context().Value("permissions").(map[string]struct{})
			for _, perm := range perms {
				if _, ok := permissionMap[perm]; !ok {
					return false
				}
			}
			return true
		},
		HasObjectPermissionFunc: func(r *http.Request, obj interface{}, perms ...string) bool {
			var permissionMap = r.Context().Value("permissions").(map[string]struct{})
			for _, perm := range perms {
				if _, ok := permissionMap[perm]; !ok {
					return false
				}
			}
			return true
		},
	}
	permissions.Tester = tester

	var request = GetRequestWithPermissions("can_view")
	if !permissions.HasPermission(request, "can_view") {
		t.Fatal("expected true, got false")
	}

	if permissions.HasPermission(request, "can_edit") {
		t.Fatal("expected false, got true")
	}
}
