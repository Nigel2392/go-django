package auth

import (
	"strings"

	"gorm.io/gorm"
)

type Group struct {
	gorm.Model
	Name        string        `gorm:"uniqueIndex;not null;size:50"`
	Description string        `gorm:"size:255"`
	Users       []*User       `gorm:"many2many:user_groups;"`
	Permissions []*Permission `gorm:"many2many:group_permissions;"`
}

// Adhere to default Admin interfaces.
//
// This is the default name of the Group model as seen in the App List.
func (g *Group) NameOf() string {
	return GROUP_MODEL_NAME
}

// Adhere to default Admin interfaces.
//
// This is the default name of the Group model's APP as seen in the App List.
func (g *Group) AppName() string {
	return AUTH_APP_NAME
}

func (g *Group) BeforeSave(tx *gorm.DB) error {
	g.Name = strings.ToLower(g.Name)
	return nil
}

func (g *Group) BeforeCreate(tx *gorm.DB) error {
	g.Name = strings.ToLower(g.Name)
	for _, permission := range g.Permissions {
		permission.Name = strings.ToLower(permission.Name)
		tx.Where("LOWER(name) = ?", permission.Name).FirstOrCreate(permission)
	}
	return nil
}

func (g *Group) Save(db *gorm.DB) error {
	return db.Where("LOWER(name) = ?", strings.ToLower(g.Name)).Save(g).Error
}

func (g *Group) Delete(db *gorm.DB) error {
	return db.Delete(g).Error
}

func (g *Group) String() string {
	return g.Name
}

func (g *Group) HasPerms(db *gorm.DB, permissions ...*Permission) bool {
	var exists bool
	db.Model(g).Where("name IN ?", permissions).Find(&exists)
	return exists
}
