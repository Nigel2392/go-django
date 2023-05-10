package auth

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-datastructures/linkedlist"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/views/fields"
)

type Permission struct {
	ID          int64            `admin-form:"readonly;disabled;omit_on_create;" gorm:"-" json:"id"`
	Name        string           `gorm:"-" json:"name"`
	Description fields.TextField `admin-form:"textarea;" gorm:"-" json:"description"`

	// Non-SQLC fields
	Groups []Group `gorm:"-" json:"groups"`
	Users  []User  `gorm:"-" json:"users"`
}

func (p *Permission) String() string {
	return p.Name
}

func (p *Permission) Save(creating bool) error {
	if creating {
		return Auth.Queries.CreatePermission(context.Background(), p)
	}
	return Auth.Queries.UpdatePermission(context.Background(), p)
}

func (p *Permission) Delete() error {
	return Auth.Queries.DeletePermission(context.Background(), p.ID)
}

func (p *Permission) StringID() string {
	return fmt.Sprintf("%d", p.ID)
}

func (p *Permission) GetFromStringID(id string) (*Permission, error) {
	var intID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return Auth.Queries.GetPermissionByID(context.Background(), intID)
}

func (u *Permission) List(page, each_page int) ([]*Permission, int64, error) {
	var count int64
	var err error
	var permissions *linkedlist.Doubly[Permission]

	permissions, err = Auth.Queries.GetPermissionsWithPagination(context.Background(), PaginationParams{
		Offset: int32((page - 1) * each_page),
		Limit:  int32(each_page),
	})
	if err != nil {
		return nil, 0, err
	}

	permissionsSlice := make([]*Permission, 0, permissions.Len())
	for permissions.Len() > 0 {
		var p = permissions.Shift()
		permissionsSlice = append(permissionsSlice, &p)
	}

	count, err = Auth.Queries.CountPermissions(context.Background())
	if err != nil {
		return nil, 0, err
	}

	return permissionsSlice, count, nil
}

// Return a new permission for a given object.
//
// The permission name will be in the format of:
//
// <typ>_<pkgPath>_<lowercase type name>
//
// If the object is not in a package, the permission name will be in the format of:
//
// <typ>_<lowercase type name>
func NewPermission(typ string, s any) *Permission {
	typeOf := reflect.TypeOf(s)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	permission := &Permission{
		Description: fields.TextField("Permission to " + typ + " an object of type " + typeOf.Name()),
	}

	if typeOf.PkgPath() != "" {
		var pkgPath = httputils.GetPkgPath(s)
		permission.Name = fmt.Sprintf("%s_%s_%s", typ, pkgPath, strings.ToLower(typeOf.Name()))
		return permission
	}

	permission.Name = fmt.Sprintf("%s_%s", typ, strings.ToLower(typeOf.Name()))
	return permission
}

// Retrieve the CREATE permission for a given object
func PermCreate(s any) *Permission {
	return NewPermission("create", s)
}

// Retrieve the VIEW/READ permission for a given object
func PermView(s any) *Permission {
	return NewPermission("view", s)
}

// Retrieve the UPDATE permission for a given object
func PermUpdate(s any) *Permission {
	return NewPermission("update", s)
}

// Retrieve the DELETE permission for a given object
func PermDelete(s any) *Permission {
	return NewPermission("delete", s)
}

// Retrieve the LIST permission for a given object
func PermList(s any) *Permission {
	return NewPermission("list", s)
}

// Returns all the permissions for a given object.
func PermAll(s any) []*Permission {
	return []*Permission{
		PermCreate(s),
		PermView(s),
		PermUpdate(s),
		PermDelete(s),
		PermList(s),
	}
}

// Chain permissions together
func PermChain(permissions ...any) []*Permission {
	var perms []*Permission
	for _, p := range permissions {
		switch p := p.(type) {
		case *Permission:
			perms = append(perms, p)
		case []*Permission:
			perms = append(perms, p...)
		default:
			panic("PermChain: this is not a permission!")
		}
	}
	return perms
}

func SuperPerm() *Permission {
	var p = &Permission{
		Name:        "*",
		Description: "Super permission, allows all actions.",
	}
	return p
}

type PermissionMap map[string]*Permission

func NewPermissionMap(s any) PermissionMap {
	pm := make(PermissionMap)
	// Initialize permissions for the given model.
	pm["create"] = PermCreate(s)
	pm["update"] = PermUpdate(s)
	pm["delete"] = PermDelete(s)
	pm["view"] = PermView(s)
	pm["list"] = PermList(s)

	return pm
}

func (pm PermissionMap) Get(name string) *Permission {
	return pm[name]
}

func (pm PermissionMap) Create() *Permission {
	return pm["create"]
}

func (pm PermissionMap) Update() *Permission {
	return pm["update"]
}

func (pm PermissionMap) Delete() *Permission {
	return pm["delete"]
}

func (pm PermissionMap) View() *Permission {
	return pm["view"]
}

func (pm PermissionMap) List() *Permission {
	return pm["list"]
}

func (pm PermissionMap) All() []*Permission {
	return []*Permission{
		pm["create"],
		pm["update"],
		pm["delete"],
		pm["view"],
		pm["list"],
	}
}
