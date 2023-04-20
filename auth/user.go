package auth

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/core/cache"
	"github.com/Nigel2392/go-django/core/models"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/go-django/core/secret"
	"github.com/Nigel2392/go-django/forms"

	"github.com/Nigel2392/go-django/forms/validators"

	"gorm.io/gorm"
)

const groups_suffix = "_groups"

// User is the default user model.
//
// It supports groups, permissions, and more.
//
// It is the default model used by the auth.Manager.
type User struct {
	models.Model    `form:"disabled" json:"-"`
	Email           string   `gorm:"uniqueIndex;not null;size:255" form:"required:true,type:email"`
	Username        string   `gorm:"uniqueIndex;not null;size:75" form:"username required:trues"`
	Password        string   `gorm:"not null;size:1024" form:"bcrypt, type:password,custom:onfocus='this.removeAttribute('readonly');'" admin:"protected" json:"-"`
	FirstName       string   `gorm:"size:50" form:"first_name"`
	LastName        string   `gorm:"size:50" form:"last_name"`
	IsLoggedIn      bool     `gorm:"-" form:"-" json:"-"`
	IsAdministrator bool     `form:"needs_admin:true"`
	IsActive        bool     `gorm:"default:true"`
	Groups          []*Group `gorm:"many2many:user_groups;"`
}

// Get the value of the currently set login field.
func (u *User) LoginField() string {
	var a, err = modelutils.GetField(u, USER_MODEL_LOGIN_FIELD, true)
	if err != nil {
		return ""
	}
	switch ret := a.(type) {
	case string:
		return ret
	default:
		return fmt.Sprintf("%v", ret)
	}
}

// Set the value of the current field used to log a user in.
func (u *User) SetLoginField(value string) error {
	return modelutils.SetField(u, USER_MODEL_LOGIN_FIELD, value)
}

// Adhere to default Admin interfaces.
//
// Allows the user to be searched by email or username.
func (u *User) AdminSearch(query string, tx *gorm.DB) *gorm.DB {
	var q = strings.TrimSpace(query)
	if len(q) == 0 {
		return tx
	}
	return tx.Where("email LIKE ? OR username LIKE ?", "%"+q+"%", "%"+q+"%")
}

// Adhere to default Admin interfaces.
//
// Allows other packages to easily get the user's page URL.
func (u *User) AbsoluteURL() string {
	if USER_ABSOLUTE_URL_FUNC != nil {
		return USER_ABSOLUTE_URL_FUNC(u)
	}
	panic("User.AbsoluteURL() not implemented. Set USER_ABSOLUTE_URL_FUNC to a function that returns a string based on the user.")
}

// Adhere to default Admin interfaces.
//
// This is the default name of the User model as seen in the App List.
func (u *User) NameOf() string {
	return USER_MODEL_NAME
}

// Adhere to default Admin interfaces.
//
// This is the default name of the User model's APP as seen in the App List.
func (u *User) AppName() string {
	return AUTH_APP_NAME
}

// Gorm hooks.
func (u *User) BeforeSave(tx *gorm.DB) error {
	if field, err := modelutils.GetField(u, USER_MODEL_LOGIN_FIELD, true); err == nil {
		if field == "" {
			return errors.New("The " + USER_MODEL_LOGIN_FIELD + " field cannot be blank.")
		}
	}
	SIGNAL_BEFORE_USER_SAVE.Send(u)
	return nil
}

// Gorm hooks.
func (u *User) AfterSave(tx *gorm.DB) error {
	SIGNAL_AFTER_USER_SAVE.Send(u)
	return nil
}

// Gorm hooks.
func (u *User) BeforeDelete(tx *gorm.DB) error {
	SIGNAL_BEFORE_USER_DELETE.Send(u)
	return nil
}

// Gorm hooks.
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	SIGNAL_BEFORE_USER_UPDATE.Send(u)
	return nil
}

// Gorm hooks.
func (u *User) AfterUpdate(tx *gorm.DB) error {
	SIGNAL_AFTER_USER_UPDATE.Send(u)
	return nil
}

// Gorm hooks.
func (u *User) AfterDelete(tx *gorm.DB) error {
	SIGNAL_AFTER_USER_DELETE.Send(u)
	return nil
}

// Gorm hooks.
// Add the user to the default group if they are not an admin.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if field, err := modelutils.GetField(u, USER_MODEL_LOGIN_FIELD, true); err == nil {
		if field == "" {
			return errors.New("The " + USER_MODEL_LOGIN_FIELD + " field cannot be blank.")
		}
	}
	//	var uuid = uuid.New()
	//	u.ID = models.DefaultIDField(uuid)
	if !u.IsAdministrator && len(DEFAULT_USER_GROUP_NAMES) > 0 {
		tx.Where("LOWER(name) IN ?", DEFAULT_USER_GROUP_NAMES).Find(&u.Groups)
	}
	SIGNAL_BEFORE_USER_CREATE.Send(u)
	return nil
}

// Gorm hooks.
func (u *User) AfterCreate(tx *gorm.DB) error {
	SIGNAL_AFTER_USER_CREATE.Send(u)
	return nil
}

// IsAdmin returns true if the user is an admin.
func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}

// IsAuthenticated returns true if the user is logged in.
func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.ID > 0
}

// Return the string representation of the user.
func (u *User) String() string {
	var b strings.Builder
	b.WriteString(u.string())
	if u.IsLoggedIn || u.ID > 0 {
		if !u.IsActive {
			b.WriteString(" (Inactive)")
		}
		if u.IsAdministrator {
			b.WriteString(" (Admin)")
		}
		if u.ID != 0 {
			b.WriteString(" (ID: " + strconv.Itoa(int(u.ID)) + ")")
		}
	}
	return b.String()
}

func (u *User) string() string {
	if u.IsLoggedIn || u.ID > 0 {
		if u.FirstName != "" && u.LastName != "" {
			return u.FirstName + " " + u.LastName
		} else if u.FirstName != "" {
			return u.FirstName
		} else if u.LastName != "" {
			return u.LastName
		} else if u.Username != "" {
			return u.Username
		} else {
			return u.Email
		}
	} else {
		return "AnonymousUser"
	}
}

// Sets a hashed password on the user instance.
// This will not automatically update the database!
func (u *User) SetPassword(password string) error {
	if err := validators.PasswordStrength(forms.NewValue(password)); err != nil {
		//lint:ignore ST1005 Password is a field.
		return errors.New("Password is too weak.")
	}
	hash, err := BcryptHash(password)
	if err != nil {
		//lint:ignore ST1005 Internal Server Error is a message.
		return errors.New("Internal Server Error")
	}
	u.Password = hash
	return nil
}

// Updates the user's password, and saves the user to the database.
func (u *User) ChangePassword(password string) error {
	if u.CheckPassword(password) == nil {
		//lint:ignore ST1005 Might be message to user.
		return errors.New("New password must be different from old password.")
	}
	var err = u.SetPassword(password)
	if err != nil {
		return err
	}
	// Update the password in the database.
	err = auth_db.Model(&User{}).Where("id = ?", u.ID).Update("password", u.Password).Error
	if err != nil {
		//lint:ignore ST1005 Could not update password.
		return errors.New("Could not update password.")
	}
	if err == nil && auth_cache != nil {
		auth_cache.Delete(hashUser(u, groups_suffix))
	}
	return nil
}

// Validate if the user's password matches the given password.
func (u *User) CheckPassword(password string) error {
	return BcryptCompare(u.Password, password)
}

func (u *User) validateFields() error {
	if err := defaultValidation(u); err != nil {
		return err
	}
	if err := validators.PasswordStrength(forms.NewValue(u.Password)); err != nil {
		return err
	}
	return nil
}

func (u *User) validate() error {
	var login_field, err = modelutils.GetField(u, USER_MODEL_LOGIN_FIELD, true)
	if err != nil {
		return err
	}
	var lflc = strings.ToLower(USER_MODEL_LOGIN_FIELD) // Login field lower case
	// Find the user by email and username, case insensitive
	var user User
	err = auth_db.Select(fmt.Sprintf("LOWER(%s) as %s", lflc, lflc)).Where(
		auth_db.Where(fmt.Sprintf("LOWER(%s) = ?", lflc), strings.ToLower(login_field.(string))),
	).First(&user).Error
	if err == nil {
		return errors.New("User already exists")
	}
	return nil
}

/*


	QUERIES


*/

// Refresh the user, the instance's groups and permissions.
func (u *User) Refresh() error {
	var err = auth_db.Model(u).Preload("Groups.Permissions").Find(&u).Error
	if err == nil && auth_cache != nil {
		auth_cache.Set(hashUser(u, groups_suffix), u.Groups, cache.DefaultExpiration)
	}
	return err
}

// Update the user
func (u *User) Update() error {
	// Get the user if it exists
	var user = &User{}
	err := auth_db.Select("LOWER(email) as email, LOWER(username) as username").Where(
		auth_db.Where("LOWER(email) = ?", strings.ToLower(u.Email)).Or("LOWER(username) = ?", strings.ToLower(u.Username)),
	).First(user).Error
	// If the does not exist, return an error
	if err != nil {
		return errors.New("user does not exist")
	}

	err = auth_db.Save(u).Error
	// Delete the cache for the user
	if auth_cache != nil {
		auth_cache.Delete(hashUser(u, groups_suffix))
	}
	return err
}

// Delete the user
func (u *User) Delete() error {
	return auth_db.Delete(u).Error
}

// Get the user by ID
// Preload the groups and permissions
func GetUserByID(id interface{}) (*User, error) {
	var user = &User{}
	err := auth_db.Preload("Groups.Permissions").Where("id = ?", id).First(user).Error
	if err != nil {
		return nil, err
	}

	if auth_cache != nil {
		auth_cache.Set(hashUser(user, groups_suffix), user.Groups, cache.DefaultExpiration)
	}

	return user, nil
}

// Groups
func (u *User) AddGroup(db *gorm.DB, groupName string) error {
	var g Group = Group{Name: strings.ToLower(groupName)}
	db.FirstOrCreate(&g, g)
	var err = db.Model(u).Association("Groups").Append(&g)
	if err != nil && auth_cache != nil {
		auth_cache.Delete(hashUser(u, groups_suffix))
	}
	return err
}

func (u *User) RemoveGroup(db *gorm.DB, groupName string) error {
	var err = db.Model(&Group{}).Delete("name = ?", groupName).Error
	if err != nil {
		return err
	}
	if auth_cache != nil {
		auth_cache.Delete(hashUser(u, groups_suffix))
	}
	return nil
}

func (u *User) SetGroups(db *gorm.DB, groups ...*Group) error {
	for _, group := range groups {
		if group.Name == "" {
			return errors.New("group name is required")
		}
		group.Name = strings.ToLower(group.Name)
		var g *Group = &Group{Name: group.Name}
		if group.ID == 0 {
			db.Where(g).FirstOrCreate(g, g)
			group = g
		}
	}
	// return db.Model(u).Association("Groups").Replace(groups)
	// Delete all groups
	err := db.Model(u).Association("Groups").Clear()
	if err != nil {
		if auth_cache != nil {
			auth_cache.Delete(hashUser(u, groups_suffix))
		}
		return err
	}

	// Add the groups
	err = db.Model(u).Association("Groups").Append(groups)
	if err != nil {
		if auth_cache != nil {
			auth_cache.Delete(hashUser(u, groups_suffix))
		}
		return err
	}
	if auth_cache != nil {
		auth_cache.Delete(hashUser(u, groups_suffix))
	}
	return nil
}

// Validate if the user has the given group
func (u *User) HasGroup(groupNames ...string) bool {
	if u.IsAdministrator {
		return true
	}
	if hasGroup(u.Groups, groupNames...) {
		return true
	}

	var groups []*Group
	if auth_cache != nil {
		var g, err = auth_cache.Get(hashUser(u, groups_suffix))
		if err == nil && g != nil {
			groups = g.Value().([]*Group)
			return hasGroup(groups, groupNames...)
		}
	}

	err := auth_db.Model(u).Association("Groups").Find(&groups)
	if err != nil {
		if auth_cache != nil {
			auth_cache.Delete(hashUser(u, groups_suffix))
		}
		return false
	}

	if auth_cache != nil {
		auth_cache.Set(hashUser(u, groups_suffix), groups, cache.DefaultExpiration)
	}

	return hasGroup(groups, groupNames...)
}

func hasGroup(groups []*Group, groupNames ...string) bool {
	var hasGroups []string
	for _, group := range groups {
		for _, groupName := range groupNames {
			if strings.EqualFold(group.Name, groupName) {
				hasGroups = append(hasGroups, groupName)
			}
		}
	}
	return len(hasGroups) == len(groupNames)
}

// Validate if the user has the given permissions
func (u *User) HasPerms(permissions ...*Permission) bool {

	// Try to refresh the user's groups.
	// Commented out to reduce queries.
	//	if len(u.Groups) == 0 { //&& auth_db.Model(u).Association("Groups").Count() > 0 {
	//		u.Refresh()
	//	}

	// If the user has no groups, or is an administrator, return u.IsAdministrator.
	if len(u.Groups) == 0 || u.IsAdministrator {
		return u.IsAdministrator
	}

	// Check if the user has the "all" permission,
	// or if the user has all the permissions.
	for _, group := range u.Groups {
		for _, perm := range group.Permissions {
			if perm.Name == "all" || perm.Name == "*" {
				return true
			}
			for i, p := range permissions {
				if strings.EqualFold(p.Name, perm.Name) {
					permissions = removeSliceItem(permissions, i)
				}
			}
		}
	}

	return len(permissions) == 0
}

// Validate if the user has the given permissions
// by their names.
func (u *User) HasStrPerms(p ...string) bool {
	if len(p) == 0 {
		return true
	}

	var perms []*Permission = make([]*Permission, 0)

	// Fetch the permissions from the database if they are not already loaded.
	if len(p) > 0 {
		var dbPerms []*Permission
		err := auth_db.Where("LOWER(name) IN (?)", p).Find(&dbPerms).Error
		if err != nil {
			return false
		}
		perms = append(perms, dbPerms...)
	}

	return u.HasPerms(perms...)
}

func sumStr(s ...string) int {
	var sum int
	for _, str := range s {
		sum += len(str)
	}
	return sum
}

func hashUser(user *User, extra ...string) string {
	var b strings.Builder

	b.Grow(sumStr(
		user.Email,
		user.Username,
	))

	b.WriteString(user.Email)
	b.WriteString(user.Username)

	var hash = secret.FnvHash(b.String()).String()
	b.Reset()
	if extrLen := sumStr(extra...); b.Cap() < (extrLen + len(hash)) {
		b.Grow((extrLen + len(hash)) - b.Cap())
	}
	b.WriteString(hash)
	for _, str := range extra {
		b.WriteString(str)
	}
	return b.String()
}
