package auth

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/core/httputils"

	"gorm.io/gorm"
)

// Default permission model.
//
// This will be used application-wide by default.
type Permission struct {
	gorm.Model
	Name        string   `gorm:"uniqueIndex;not null;size:50"`
	Description string   `gorm:"size:255"`
	Groups      []*Group `gorm:"many2many:group_permissions;"`
}

// Adhere to default Admin interfaces.
//
// This is the default name of the Permission model as seen in the App List.
func (p *Permission) NameOf() string {
	return PERMISSION_MODEL_NAME
}

// Adhere to default Admin interfaces.
//
// This is the default name of the Permission model's APP as seen in the App List.
func (p *Permission) AppName() string {
	return AUTH_APP_NAME
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
		Description: "Permission to " + typ + " an object of type " + typeOf.Name(),
	}

	if typeOf.PkgPath() != "" {
		var pkgPath = httputils.GetPkgPath(s)
		permission.Name = fmt.Sprintf("%s_%s_%s", typ, pkgPath, strings.ToLower(typeOf.Name()))
		return permission
	}

	permission.Name = fmt.Sprintf("%s_%s", typ, strings.ToLower(typeOf.Name()))
	return permission
}

func (p *Permission) String() string {
	return p.Name
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

// Save the permission to the database.
func (p *Permission) Save(db *gorm.DB) error {
	return db.FirstOrCreate(p, "name = ?", p.Name).Error
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
