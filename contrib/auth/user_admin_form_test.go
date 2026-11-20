//go:build test
// +build test

package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	_ "github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

func buildUserEditAdminForm(req *http.Request, user *auth.User) *admin.AdminForm[modelforms.ModelForm[*auth.User], *auth.User] {
	mf := modelforms.NewBaseModelForm(req.Context(), user)
	mf.SetExclude("ID", "Password", "CreatedAt", "UpdatedAt", "LastLogin")

	// Mirrors contrib/auth/app.go User EditView panels (which include TitlePanel / RowPanel,
	// triggering AdminForm.FormSet() -> a.Instance() during AdminForm.SaveFunc()).
	panels := []admin.Panel{
		admin.TitlePanel(
			admin.FieldPanel("Email"),
		),
		admin.FieldPanel("Username"),
		admin.RowPanel(
			admin.FieldPanel("FirstName"),
			admin.FieldPanel("LastName"),
		),
		admin.FieldPanel("IsAdministrator"),
		admin.FieldPanel("IsActive"),
		admin.FieldPanel("Permissions"),
		admin.FieldPanel("Groups"),
	}

	var iface modelforms.ModelForm[*auth.User] = mf
	af := admin.NewAdminForm(req, iface, panels...)
	af.Load()
	return af
}

func TestUserAdminForm_EditPermissionsAndGroupsMultiSelect(t *testing.T) {
	ctx := t.Context()

	cleanup := func() {
		_, _ = queries.GetQuerySet(&users.UserPermission{}).Delete()
		_, _ = queries.GetQuerySet(&users.UserGroup{}).Delete()
		_, _ = queries.GetQuerySet(&auth.User{}).Filter("Username", "adminformuser").Delete()
		_, _ = queries.GetQuerySet(&users.Permission{}).Filter("Name__in",
			"adminform:settings:view",
			"adminform:settings:edit",
			"adminform:users:delete",
		).Delete()
		_, _ = queries.GetQuerySet(&users.Group{}).Filter("Name__in",
			"adminform_editors",
			"adminform_viewers",
		).Delete()
	}
	cleanup()
	t.Cleanup(cleanup)

	pView := models.Setup(ctx, &users.Permission{
		Name:        "adminform:settings:view",
		Description: "View settings",
	})
	pEdit := models.Setup(ctx, &users.Permission{
		Name:        "adminform:settings:edit",
		Description: "Edit settings",
	})
	pDelete := models.Setup(ctx, &users.Permission{
		Name:        "adminform:users:delete",
		Description: "Delete users",
	})
	for _, perm := range []*users.Permission{pView, pEdit, pDelete} {
		if _, err := queries.GetQuerySet(&users.Permission{}).Create(perm); err != nil {
			t.Fatalf("create permission %q: %v", perm.Name, err)
		}
	}

	gEditors := models.Setup(ctx, &users.Group{Name: "adminform_editors"})
	gViewers := models.Setup(ctx, &users.Group{Name: "adminform_viewers"})
	for _, grp := range []*users.Group{gEditors, gViewers} {
		if _, err := queries.GetQuerySet(&users.Group{}).Create(grp); err != nil {
			t.Fatalf("create group %q: %v", grp.Name, err)
		}
	}

	u := models.Setup(ctx, &auth.User{
		Email:     drivers.MustParseEmail("adminformuser@example.com"),
		Username:  "adminformuser",
		FirstName: "Admin",
		LastName:  "Form",
		Password:  auth.NewPassword("Secret123!"),
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	})
	createdUser, err := auth.GetUserQuerySet().WithContext(ctx).Create(u)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	reloadUser := func(t *testing.T) *auth.User {
		t.Helper()
		row, err := auth.GetUserQuerySet().WithContext(ctx).Filter("ID", createdUser.ID).First()
		if err != nil {
			t.Fatalf("reload user: %v", err)
		}
		return row.Object
	}

	permNames := func(t *testing.T, u *auth.User) []string {
		t.Helper()
		rows, err := u.Permissions.Objects().WithContext(ctx).All()
		if err != nil {
			t.Fatalf("load user permissions: %v", err)
		}
		names := make([]string, 0, len(rows))
		for _, r := range rows {
			names = append(names, r.Object.Name)
		}
		sort.Strings(names)
		return names
	}

	groupNames := func(t *testing.T, u *auth.User) []string {
		t.Helper()
		rows, err := u.Groups.Objects().WithContext(ctx).All()
		if err != nil {
			t.Fatalf("load user groups: %v", err)
		}
		names := make([]string, 0, len(rows))
		for _, r := range rows {
			names = append(names, r.Object.Name)
		}
		sort.Strings(names)
		return names
	}

	submitUserEdit := func(t *testing.T, u *auth.User, permIDs []uint64, groupIDs []uint64) *admin.AdminForm[modelforms.ModelForm[*auth.User], *auth.User] {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/admin/auth/user/edit", nil)
		af := buildUserEditAdminForm(req, u)

		data := url.Values{
			"Email":     {"adminformuser@example.com"},
			"Username":  {"adminformuser"},
			"FirstName": {"Admin"},
			"LastName":  {"Form"},
			"IsActive":  {"on"},
		}
		for _, id := range permIDs {
			data.Add("Permissions", strconv.FormatUint(id, 10))
		}
		for _, id := range groupIDs {
			data.Add("Groups", strconv.FormatUint(id, 10))
		}
		af.WithData(data, nil, req)

		if !forms.IsValid(req.Context(), af) {
			t.Fatalf("expected User AdminForm valid, errList=%v boundErrs=%v", af.ErrorList(), af.BoundErrors())
		}
		if _, err := af.Save(); err != nil {
			var bErrs = make(map[string][]error)
			if be := af.BoundErrors(); be != nil {
				for h := be.Front(); h != nil; h = h.Next() {
					bErrs[h.Key] = h.Value
				}
			}
			t.Fatalf("AdminForm.Save() failed: %v (errList=%v boundErrs=%v)", err, af.ErrorList(), bErrs)
		}
		if len(af.ErrorList()) > 0 || (af.BoundErrors() != nil && af.BoundErrors().Len() > 0) {
			t.Fatalf("expected zero errors after Save(), errList=%v boundErrs=%v", af.ErrorList(), af.BoundErrors())
		}
		return af
	}

	t.Run("AddPermissions", func(t *testing.T) {
		user := reloadUser(t)
		submitUserEdit(t, user, []uint64{pView.ID, pEdit.ID}, nil)

		reloaded := reloadUser(t)
		gotPerms := permNames(t, reloaded)
		wantPerms := []string{"adminform:settings:edit", "adminform:settings:view"}
		if len(gotPerms) != len(wantPerms) || gotPerms[0] != wantPerms[0] || gotPerms[1] != wantPerms[1] {
			t.Fatalf("persisted permissions = %v, want %v", gotPerms, wantPerms)
		}

		checkCtx := context.Background()
		if !reloaded.HasObjectPermission(checkCtx, nil, "adminform:settings:view", "adminform:settings:edit") {
			t.Errorf("expected HasObjectPermission to return true for added permissions")
		}
		if reloaded.HasObjectPermission(checkCtx, nil, "adminform:users:delete") {
			t.Errorf("expected HasObjectPermission to return false for unselected permission")
		}
	})

	t.Run("ReplacePermissions", func(t *testing.T) {
		user := reloadUser(t)
		submitUserEdit(t, user, []uint64{pDelete.ID}, nil)

		reloaded := reloadUser(t)
		gotPerms := permNames(t, reloaded)
		wantPerms := []string{"adminform:users:delete"}
		if len(gotPerms) != len(wantPerms) || gotPerms[0] != wantPerms[0] {
			t.Fatalf("persisted permissions = %v, want %v", gotPerms, wantPerms)
		}
	})

	t.Run("ClearPermissionsOmitted", func(t *testing.T) {
		user := reloadUser(t)
		submitUserEdit(t, user, nil, nil)

		reloaded := reloadUser(t)
		gotPerms := permNames(t, reloaded)
		if len(gotPerms) != 0 {
			t.Fatalf("expected 0 permissions after omitted submit, got %v", gotPerms)
		}
	})

	t.Run("AddAndClearGroups", func(t *testing.T) {
		user := reloadUser(t)
		submitUserEdit(t, user, nil, []uint64{gEditors.ID})

		reloaded := reloadUser(t)
		gotGroups := groupNames(t, reloaded)
		if len(gotGroups) != 1 || gotGroups[0] != "adminform_editors" {
			t.Fatalf("persisted groups = %v, want [adminform_editors]", gotGroups)
		}

		submitUserEdit(t, reloaded, nil, nil)
		reloaded = reloadUser(t)
		gotGroups = groupNames(t, reloaded)
		if len(gotGroups) != 0 {
			t.Fatalf("expected 0 groups after omitted submit, got %v", gotGroups)
		}
	})
}
