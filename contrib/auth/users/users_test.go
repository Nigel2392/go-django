package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
)

type stubAuthUser struct {
	authenticated bool
	admin         bool
}

func (u *stubAuthUser) IsAuthenticated() bool { return u.authenticated }
func (u *stubAuthUser) IsAdmin() bool         { return u.admin }

type stubTesterUser struct {
	stubAuthUser
	called bool
	allow  bool
}

func (u *stubTesterUser) HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool {
	u.called = true
	return u.allow
}

type stubBaseModelUser struct {
	Base
}

func (u *stubBaseModelUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, u, u.Base.Fields(u))
}

func reqWithUser(u authentication.User) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if u == nil {
		return req
	}
	return req.WithContext(authentication.ContextWithUser(req.Context(), u))
}

func TestBaseMethods(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	inactive := &Base{IsLoggedIn: true, IsAdministrator: true, IsActive: false, LastLogin: now}
	if inactive.IsAuthenticated() {
		t.Errorf("expected inactive user IsAuthenticated() to be false")
	}
	if inactive.IsAdmin() {
		t.Errorf("expected inactive admin IsAdmin() to be false")
	}
	if !inactive.LastLoggedIn().Equal(now) {
		t.Errorf("LastLoggedIn() = %v, want %v", inactive.LastLoggedIn(), now)
	}

	loggedOut := &Base{IsLoggedIn: false, IsAdministrator: true, IsActive: true}
	if loggedOut.IsAuthenticated() {
		t.Errorf("expected logged out user IsAuthenticated() to be false")
	}
	if !loggedOut.IsAdmin() {
		t.Errorf("expected active administrator IsAdmin() to be true")
	}

	activeAdmin := &Base{IsLoggedIn: true, IsAdministrator: true, IsActive: true}
	if !activeAdmin.IsAuthenticated() || !activeAdmin.IsAdmin() {
		t.Errorf("expected active logged-in admin to return true for both IsAuthenticated and IsAdmin")
	}
	if !activeAdmin.HasObjectPermission(context.Background(), nil, "any:perm") {
		t.Errorf("expected active admin to have any permission")
	}

	normal := &Base{IsLoggedIn: true, IsAdministrator: false, IsActive: true}
	if !normal.HasObjectPermission(context.Background(), nil) {
		t.Errorf("expected empty permission slice to return true")
	}
}

func TestGroupAndPermissionStringAndThroughDefs(t *testing.T) {
	ctx := context.Background()
	g := &Group{Name: "Editors"}
	if g.String() != "Editors" {
		t.Errorf("Group.String() = %q, want %q", g.String(), "Editors")
	}

	p := &Permission{Name: "articles:publish"}
	if p.String() != "articles:publish" {
		t.Errorf("Permission.String() = %q, want %q", p.String(), "articles:publish")
	}

	ug := &UserGroup{}
	if attrs.Define(ctx, ug).TableName() != "auth_user_groups" {
		t.Errorf("UserGroup table = %q, want auth_user_groups", attrs.Define(ctx, ug).TableName())
	}
	if len(ug.UniqueTogether()) != 1 {
		t.Errorf("expected UserGroup UniqueTogether")
	}

	up := &UserPermission{}
	if attrs.Define(ctx, up).TableName() != "auth_user_permissions" {
		t.Errorf("UserPermission table = %q, want auth_user_permissions", attrs.Define(ctx, up).TableName())
	}
	if len(up.UniqueTogether()) != 1 {
		t.Errorf("expected UserPermission UniqueTogether")
	}

	gp := &GroupPermission{}
	if attrs.Define(ctx, gp).TableName() != "auth_group_permissions" {
		t.Errorf("GroupPermission table = %q, want auth_group_permissions", attrs.Define(ctx, gp).TableName())
	}
	if len(gp.UniqueTogether()) != 1 {
		t.Errorf("expected GroupPermission UniqueTogether")
	}
}

func TestPermissionsBackend_UnitAndEdgeCases(t *testing.T) {
	pb := NewPermissionsBackend()

	if pb.HasPermission(reqWithUser(nil), "perm:read") {
		t.Errorf("expected false when user is nil")
	}

	if !pb.HasObjectPermission(reqWithUser(&stubAuthUser{authenticated: false}), nil) {
		t.Errorf("expected true when perms slice is empty and user is non-nil")
	}

	if pb.HasPermission(reqWithUser(&stubAuthUser{authenticated: false, admin: true}), "perm:read") {
		t.Errorf("expected false when user is not authenticated even if admin=true")
	}

	if !pb.HasPermission(reqWithUser(&stubAuthUser{authenticated: true, admin: true}), "perm:read") {
		t.Errorf("expected true when user is authenticated and admin")
	}

	tester := &stubTesterUser{stubAuthUser: stubAuthUser{authenticated: true}, allow: true}
	if !pb.HasObjectPermission(reqWithUser(tester), "obj", "perm:edit") || !tester.called {
		t.Errorf("expected PermissionsBackend to delegate to hasObjectPermissionsTester")
	}

	baseUser := &stubBaseModelUser{Base: Base{IsLoggedIn: true, IsActive: true}}
	baseUser.Permissions = &queries.RelM2M[*Permission, *UserPermission]{}
	baseUser.Groups = &queries.RelM2M[*Group, *UserGroup]{}
	ctx := cacheFlags(context.Background(), false, true)
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(authentication.ContextWithUser(ctx, baseUser))
	if pb.HasObjectPermission(r, nil, "missing:perm") {
		t.Errorf("expected false when baseUser lacks permission and DB check disabled")
	}

	t.Run("UnsupportedUserPanics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for user not implementing hasObjectPermissionsTester or baseModelGetter")
			}
		}()
		pb.HasPermission(reqWithUser(&stubAuthUser{authenticated: true, admin: false}), "perm:read")
	})

	t.Run("CheckDatabasePermissionsPanicsOnNilBackref", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic when Base.backref is nil")
			}
		}()
		b := &Base{IsLoggedIn: true, IsActive: true}
		b.checkDatabasePermissions(context.Background(), nil, "perm:read")
	})

	t.Run("CheckDatabasePermissionsPanicsOnMissingPrimary", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic when backref has no primary field")
			}
		}()
		b := &stubBaseModelUser{Base: Base{IsLoggedIn: true, IsActive: true}}
		b.backref = b
		b.checkDatabasePermissions(context.Background(), nil, "perm:read")
	})
}
