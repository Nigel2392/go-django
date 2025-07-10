package users

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "unsafe"
)

type Base struct {
	IsAdministrator bool `json:"is_administrator" attrs:"blank"`
	IsActive        bool `json:"is_active" attrs:"blank;default=true"`
	IsLoggedIn      bool `json:"is_logged_in"`

	Groups      *queries.RelM2M[*Group, *UserGroup]           `json:"-"`
	Permissions *queries.RelM2M[*Permission, *UserPermission] `json:"-"`
	backref     attrs.Definer
}

func (u *Base) Fields(user attrs.Definer) []any {

	u.backref = user

	return []any{
		attrs.NewField(user, "IsAdministrator", &attrs.FieldConfig{
			Column: "is_administrator",
		}),
		attrs.NewField(user, "IsActive", &attrs.FieldConfig{
			Column: "is_active",
		}),
		fields.NewManyToManyField[*queries.RelM2M[*Group, *UserGroup]](
			user, "Groups", &fields.FieldConfig{
				ScanTo:            &u.Groups,
				ReverseName:       "UserGroups",
				NoReverseRelation: true,
				Rel: attrs.Relate(
					&Group{}, "",
					&attrs.ThroughModel{
						This:   &UserGroup{},
						Source: "UserID",
						Target: "GroupID",
					},
				),
			},
		),
		fields.NewManyToManyField[*queries.RelM2M[*Permission, *UserPermission]](
			user, "Permissions", &fields.FieldConfig{
				ScanTo:            &u.Permissions,
				ReverseName:       "UserPermissions",
				NoReverseRelation: true,
				Rel: attrs.Relate(
					&Permission{}, "",
					&attrs.ThroughModel{
						This:   &UserPermission{},
						Source: "UserID",
						Target: "PermissionID",
					},
				),
			},
		),
	}
}

func (u *Base) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *Base) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}

type permsCheckContextKey struct{}

type permsCheckFlags struct {
	disableCache   bool
	disableDbCheck bool
}

//go:linkname cacheFlags
func cacheFlags(ctx context.Context, disableCache bool, disableDB bool) context.Context { //lint:ignore U1000 this function is exported with linkname for testing purposes
	return context.WithValue(ctx, permsCheckContextKey{}, permsCheckFlags{
		disableCache:   disableCache,
		disableDbCheck: disableDB,
	})
}

// checkCachedPermissions checks the user's permissions cache and the group permissions cache.
// It returns a slice of permissions that the user does not have cached.
// If the user has all permissions cached, it returns an empty slice and the user is considered to have those permissions.
func (u *Base) checkCachedPermissions(ctx context.Context, perms ...string) []string {
	// First we check the direct user permission cache.
	var pCache = u.Permissions.Cache()
	if pCache != nil && pCache.Len() > 0 {
		var hasNot = make([]string, 0, len(perms))
		var permMap = make(map[string]struct{}, pCache.Len())
		for head := pCache.Front(); head != nil; head = head.Next() {
			permMap[head.Value.Object.Name] = struct{}{}
		}

		for _, perm := range perms {
			if _, ok := permMap[perm]; !ok {
				hasNot = append(hasNot, perm)
			}
		}

		if len(hasNot) == 0 {
			return []string{}
		}

		perms = hasNot
	}

	// We can check the group cache afterwards.
	var gCache = u.Groups.Cache()
	if gCache != nil && gCache.Len() > 0 {
		for head := gCache.Front(); head != nil; head = head.Next() {
			var group = head.Value.Object
			var pCache = group.Permissions.Cache()
			if pCache != nil && pCache.Len() > 0 {
				var hasNot = make([]string, 0, len(perms))
				var permMap = make(map[string]struct{}, pCache.Len())
				for head := pCache.Front(); head != nil; head = head.Next() {
					permMap[head.Value.Object.Name] = struct{}{}
				}

				for _, perm := range perms {
					if _, ok := permMap[perm]; !ok {
						hasNot = append(hasNot, perm)
					}
				}

				if len(hasNot) == 0 {
					return []string{}
				}

				perms = hasNot
			}

			if len(perms) == 0 {
				return []string{}
			}
		}
	}

	return perms
}

func (u *Base) checkDatabasePermissions(ctx context.Context, _ interface{}, perms ...string) bool {
	if u.backref == nil {
		panic("Base backref is not set, cannot check permissions")
	}

	var q = make([]expr.Expression, 0, len(perms)*2)
	var defs = u.backref.FieldDefs()
	var primary = defs.Primary()
	if primary == nil {
		panic("Base backref does not have a primary field, cannot check permissions")
	}

	for _, perm := range perms {

		directPerms := queries.Objects(&Permission{}).
			Select("ID").
			Filter("Name", perm).
			Filter("ID__in", queries.Objects(&UserPermission{}).
				Select("PermissionID").
				Filter("UserID", defs.Primary().GetValue()),
			)

		groupPerms := queries.Objects(&Permission{}).
			Select("ID").
			Filter("Name", perm).
			Filter("ID__in", queries.Objects(&GroupPermission{}).
				Select("PermissionID").
				Filter("GroupID__in", queries.Objects(&UserGroup{}).
					Select("GroupID").
					Filter("UserID", defs.Primary().GetValue()),
				),
			)

		q = append(q, expr.Or(
			expr.EXISTS(directPerms),
			expr.EXISTS(groupPerms),
		))
	}

	var exists, err = queries.GetQuerySetWithContext(ctx, u.backref).
		Filter(expr.Q("ID", defs.Primary().GetValue()), expr.And(q...)).
		Exists()
	if err != nil {
		logger.Warnf("users.Base.checkDatabasePermissions: %v", err)
		return false
	}
	return exists
}

func (u *Base) HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool {

	if u.IsAdmin() || len(perms) == 0 {
		return true
	}

	var flags permsCheckFlags
	if value, ok := ctx.Value(permsCheckContextKey{}).(permsCheckFlags); ok {
		flags = value
	}

	if !flags.disableCache {
		perms = u.checkCachedPermissions(ctx, perms...)
	}

	if len(perms) == 0 {
		return true
	}

	if !flags.disableDbCheck {
		return u.checkDatabasePermissions(ctx, obj, perms...)
	}

	return false
}
