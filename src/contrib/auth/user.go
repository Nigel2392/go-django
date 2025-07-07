package auth

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	django_models "github.com/Nigel2392/go-django/src/models"
)

var (
	_ django_models.ContextSaver = (*User)(nil)
	_ queries.ActsBeforeSave     = (*User)(nil)
	_ queries.ActsBeforeCreate   = (*User)(nil)
)

type UserGroup struct {
	UserID  drivers.Uint `json:"user_id" attrs:"primary;readonly"`
	GroupID drivers.Uint `json:"group_id" attrs:"primary;readonly"`
}

type UserQuerySet struct {
	*queries.WrappedQuerySet[*User, *UserQuerySet, *queries.QuerySet[*User]]
}

// var _ queries.QuerySetCanAfterExec = (*UserQuerySet)(nil)
//
//	func (uq *UserQuerySet) PrefetchGroups() *UserQuerySet {
//		uq.prefetchGroups = true
//		return uq
//	}
//
//	func (uq *UserQuerySet) PrefetchPermissions() *UserQuerySet {
//		uq.prefetchPermissions = true
//		return uq
//	}
//
//	func (uq *UserQuerySet) handlePrefetchSingleRow(row *queries.Row[*User]) error {
//		return nil
//	}
//
//	func (uq *UserQuerySet) handlePrefetchRows(rows queries.Rows[*User]) error {
//		return nil
//	}
//
//	func (uq *UserQuerySet) AfterExec(res any) error {
//
//		switch r := res.(type) {
//		case queries.Rows[*User]:
//			return uq.handlePrefetchRows(r)
//		case *queries.Row[*User]:
//			return uq.handlePrefetchSingleRow(r)
//		}
//
//		return nil
//	}

func GetUserQuerySet() *UserQuerySet {
	userQuerySet := &UserQuerySet{}
	userQuerySet.WrappedQuerySet = queries.WrapQuerySet(
		queries.GetQuerySet[*User](&User{}),
		userQuerySet,
	)
	return userQuerySet
}

func (up *UserGroup) FieldDefs() attrs.Definitions {
	return attrs.Define(up,
		attrs.Unbound("UserID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "user_id",
		}),
		attrs.Unbound("GroupID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "group_id",
		}),
	).WithTableName("auth_user_groups")
}

func (up *UserGroup) UniqueTogether() [][]string {
	return [][]string{
		{"UserID", "GroupID"},
	}
}

type UserPermission struct {
	UserID       drivers.Uint `json:"user_id" attrs:"primary;readonly"`
	PermissionID drivers.Uint `json:"permission_id" attrs:"primary;readonly"`
}

func (up *UserPermission) FieldDefs() attrs.Definitions {
	return attrs.Define(up,
		attrs.Unbound("UserID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "user_id",
		}),
		attrs.Unbound("PermissionID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "permission_id",
		}),
	).WithTableName("auth_user_permissions")
}

func (up *UserPermission) UniqueTogether() [][]string {
	return [][]string{
		{"UserID", "PermissionID"},
	}
}

type User struct {
	models.Model    `table:"auth_users" json:"-"`
	ID              uint64           `json:"id" attrs:"primary;readonly"`
	CreatedAt       drivers.DateTime `json:"created_at" attrs:"readonly"`
	UpdatedAt       drivers.DateTime `json:"updated_at" attrs:"readonly"`
	Email           *drivers.Email   `json:"email"`
	Username        string           `json:"username"`
	Password        *Password        `json:"password"`
	FirstName       string           `json:"first_name"`
	LastName        string           `json:"last_name"`
	IsAdministrator bool             `json:"is_administrator" attrs:"blank"`
	IsActive        bool             `json:"is_active" attrs:"blank;default=true"`
	IsLoggedIn      bool             `json:"is_logged_in"`

	Groups      *queries.RelM2M[*Group, *UserGroup]           `json:"-"`
	Permissions *queries.RelM2M[*Permission, *UserPermission] `json:"-"`
}

func (u *User) String() string {
	return u.Username
}

func (u *User) SetPassword(password string) *User {
	u.Password = NewPassword(password)
	return u
}

func (u *User) BeforeCreate(ctx context.Context) error {
	u.CreatedAt = drivers.CurrentDateTime()
	return nil
}

func (u *User) BeforeSave(ctx context.Context) error {

	if u.Password.IsZero() {
		return errors.ValueError.Wrap("password cannot be empty")
	}

	u.UpdatedAt = drivers.CurrentDateTime()
	return nil
}

func (u *User) Fields() []any {
	return []any{
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
		}),
		attrs.Unbound("Email", &attrs.FieldConfig{
			Column:    "email",
			MaxLength: 255,
			MinLength: 3,
		}),
		attrs.Unbound("Username", &attrs.FieldConfig{
			Column:    "username",
			MaxLength: 16,
			MinLength: 3,
		}),
		attrs.Unbound("FirstName", &attrs.FieldConfig{
			Column:    "first_name",
			MaxLength: 75,
		}),
		attrs.Unbound("LastName", &attrs.FieldConfig{
			Column:    "last_name",
			MaxLength: 75,
		}),
		attrs.Unbound("Password", &attrs.FieldConfig{
			Column:    "password",
			MaxLength: 255,
		}),
		attrs.Unbound("IsAdministrator", &attrs.FieldConfig{
			Column: "is_administrator",
		}),
		attrs.Unbound("IsActive", &attrs.FieldConfig{
			Column: "is_active",
		}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{
			Column: "created_at",
		}),
		attrs.Unbound("UpdatedAt", &attrs.FieldConfig{
			Column: "updated_at",
		}),
		fields.NewManyToManyField[*queries.RelM2M[*Group, *UserGroup]](
			u, "Groups", &fields.FieldConfig{
				ScanTo:      &u.Groups,
				ReverseName: "UserGroups",
				ColumnName:  "",
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
			u, "Permissions", &fields.FieldConfig{
				ScanTo:      &u.Permissions,
				ReverseName: "UserPermissions",
				ColumnName:  "",
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

func (u *User) FieldDefs() attrs.Definitions {
	return u.Model.Define(u, u.Fields)
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}

func (u *User) HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool {

	if u.IsAdmin() || len(perms) == 0 {
		return true
	}

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
			return true
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
					return true
				}

				perms = hasNot
			}

			if len(perms) == 0 {
				return true
			}
		}
	}

	var q = make([]expr.Expression, 0, len(perms)*2)
	for _, perm := range perms {

		directPerms := queries.Objects[*Permission](&Permission{}).
			Select("ID").
			Filter("Name", perm).
			Filter("ID__in", queries.Objects(&UserPermission{}).
				Select("PermissionID").
				Filter("UserID", u.ID),
			)

		groupPerms := queries.Objects(&Permission{}).
			Select("ID").
			Filter("Name", perm).
			Filter("ID__in", queries.Objects(&GroupPermission{}).
				Select("PermissionID").
				Filter("GroupID__in", queries.Objects(&UserGroup{}).
					Select("GroupID").
					Filter("UserID", u.ID),
				),
			)

		q = append(q, expr.Or(
			expr.EXISTS(directPerms),
			expr.EXISTS(groupPerms),
		))
	}

	var exists, err = GetUserQuerySet().
		WithContext(ctx).
		Filter(expr.Q("ID", u.ID), expr.And(q...)).
		Exists()
	if err != nil {
		return false
	}

	return exists
}
