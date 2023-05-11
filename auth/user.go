package auth

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/views/fields"
	"github.com/Nigel2392/go-django/core/views/interfaces"
)

var (
	USER_MODEL_LOGIN_FIELD string = "Username"
)

type User struct {
	ID              int64         `admin-form:"readonly;disabled;omit_on_create;" gorm:"-" json:"id"`
	CreatedAt       time.Time     `gorm:"-" json:"created_at"`
	UpdatedAt       time.Time     `gorm:"-" json:"updated_at"`
	Email           EmailField    `gorm:"-" json:"email"`
	Username        string        `gorm:"-" json:"username"`
	Password        PasswordField `gorm:"-" json:"password"`
	FirstName       string        `gorm:"-" json:"first_name"`
	LastName        string        `gorm:"-" json:"last_name"`
	IsAdministrator bool          `gorm:"-" json:"is_administrator"`
	IsActive        bool          `gorm:"-" json:"is_active"`

	// Non-SQLC fields
	IsLoggedIn    bool                             `json:"-" gorm:"-"`
	GroupSelect   fields.DoubleMultipleSelectField `gorm:"-" json:"group_select"`
	UploadAnImage fields.FileField                 `gorm:"-" json:"image"`
	Groups        []Group                          `json:"groups" gorm:"-"`
	Permissions   []Permission                     `json:"permissions" gorm:"-"`

	// Used for forms by other packages.
	SelectedAsOption bool `gorm:"-" json:"-"`
}

var LABELFUNC = func(u *User) string {
	return u.LoginField()
}

func (u *User) OptionLabel() string {
	return LABELFUNC(u)
}

func (u *User) OptionValue() string {
	return strconv.FormatInt(u.ID, 10)
}

func (u *User) OptionSelected() bool {
	return u.SelectedAsOption
}

func (u *User) Save(creating bool) error {
	var err error
	if creating {
		err = Auth.Queries.CreateUser(context.Background(), u)
	} else {
		err = Auth.Queries.UpdateUser(context.Background(), u)
	}

	if err != nil {
		return err
	}

	var groups = u.GroupSelect.Left
	var groupIDs []int64
	if len(groups) > 0 {
		groupIDs = make([]int64, len(groups))
		for i, group := range groups {
			var intID, err = strconv.ParseInt(group.OptionValue(), 10, 64)
			if err != nil {
				return err
			}
			groupIDs[i] = intID
		}
	}

	return Auth.Queries.OverrideUserGroups(context.Background(), u.ID, groupIDs)
}

func (u *User) GetGroupSelectLabel() string {
	return "Groups"
}

func (u *User) FormValues(s []string) error {
	if len(s) <= 0 {
		return nil
	}

	var intID, err = strconv.ParseInt(s[0], 10, 64)
	if err != nil {
		return errors.New("could not convert ID to int64")
	}

	user, err := Auth.Queries.GetUserByID(context.Background(), intID)
	if err != nil {

		return err
	}
	*u = *user
	return nil
}

func (u *User) GetGroupSelectOptions() (thisOptions, otherOptions []interfaces.Option) {
	var userGroups, err = Auth.Queries.GetGroupsByUserID(context.Background(), u.ID)
	if err != nil {
		return nil, nil
	}
	var allGroups, err2 = Auth.Queries.GroupsNotInUser(context.Background(), u.ID)
	if err2 != nil {
		return nil, nil
	}
	thisOptions = make([]interfaces.Option, 0, userGroups.Len())
	otherOptions = make([]interfaces.Option, 0, allGroups.Len())
	for n := userGroups.Head(); n != nil; n = n.Next() {
		var g = n.Value()
		var ptrG = &g
		thisOptions = append(thisOptions, fields.Option{
			Val:  ptrG.StringID(),
			Text: g.Name,
		})
	}
	for n := allGroups.Head(); n != nil; n = n.Next() {
		var g = n.Value()
		var ptrG = &g
		otherOptions = append(otherOptions, fields.Option{
			Val:  ptrG.StringID(),
			Text: g.Name,
		})
	}

	return thisOptions, otherOptions
}

func (u *User) Delete() error {
	return Auth.Queries.DeleteUser(context.Background(), u.ID)
}

func (p *User) GetFromStringID(id string) (*User, error) {
	var intID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	return Auth.Queries.GetUserByID(context.Background(), intID)
}

func (p *User) StringID() string {
	return fmt.Sprintf("%d", p.ID)
}

func (u *User) List(page, each_page int) ([]*User, int64, error) {
	var count int64
	var users, err = Auth.Queries.GetUsersWithPagination(context.Background(), PaginationParams{
		Offset: int32((page - 1) * each_page),
		Limit:  int32(each_page),
	})
	if err != nil {
		return nil, 0, err
	}
	var usersSlice []*User = make([]*User, 0, users.Len())
	for users.Len() > 0 {
		var user = users.Shift()
		usersSlice = append(usersSlice, &user)
	}

	count, err = Auth.Queries.CountUsers(context.Background())
	if err != nil {
		return nil, 0, err
	}
	return usersSlice, count, nil
}

// Scanner and valuer for the User struct
func (u *User) Scan(value interface{}) error {
	if u == nil {
		u = &User{}
	}
	switch v := value.(type) {
	case int8:
		u.ID = int64(v)
	case int16:
		u.ID = int64(v)
	case int32:
		u.ID = int64(v)
	case int64:
		u.ID = v
	case int:
		u.ID = int64(v)
	case []byte:
		var err error
		u.ID, err = strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return err
		}
	case string:
		var err error
		u.ID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
	default:
		return errors.New("Invalid type for User.ID")
	}
	return nil
}

func (u *User) Value() (driver.Value, error) {
	return u.ID, nil
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
			return string(u.Email)
		}
	} else {
		return "AnonymousUser"
	}
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn
}

// Get the value of the currently set login field.
func (u *User) LoginField() string {
	var v = reflect.ValueOf(u)
	v = reflect.Indirect(v)
	var rField = v.FieldByName(USER_MODEL_LOGIN_FIELD)
	if !rField.IsValid() {
		return ""
	}
	var iFace = rField.Interface()
	switch answer := iFace.(type) {
	case string:
		return answer
	default:
		return fmt.Sprintf("%v", answer)
	}
}

// Set the value of the current field used to log a user in.
func (u *User) SetLoginField(value string) error {
	var v = reflect.ValueOf(u)
	v = reflect.Indirect(v)
	var rField = v.FieldByName(USER_MODEL_LOGIN_FIELD)
	if !rField.IsValid() {
		return errors.New("Invalid login field")
	}
	var iFace = rField.Interface()
	switch iFace.(type) {
	case string:
		rField.SetString(value)
		return nil
	case int, int8, int16, int32, int64:
		var i, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		rField.SetInt(i)
		return nil
	}
	return errors.New("Invalid login field")
}

type WithDB struct {
	DB  AuthQuerier
	CTX context.Context
}

// Validate if the user has the given permissions
//
// It will not fetch any permissions from the database.
func (u *User) HasPermissions(permissions ...string) bool {

	// If the user has no groups, or is an administrator, return u.IsAdministrator.
	if u.IsAdministrator {
		return true
	}

	// Make a map of permission names to booleans.
	var permMap = make(map[string]bool)
	for _, perm := range permissions {
		permMap[perm] = false
	}

	// Loop through the user's groups and check if they have the permissions.
	//
	// If the user has the "all" or "*" permission, they have all permissions, and we can return true.
	for _, group := range u.Groups {
		for _, perm := range group.Permissions {
			if perm.Name == "all" || perm.Name == "*" {
				return true
			}
			if _, ok := permMap[perm.Name]; ok {
				permMap[perm.Name] = true
			}
		}
	}

	// Loop through the map of permissions and check if they are all true.
	for _, v := range permMap {
		if !v {
			return false
		}
	}

	return true
}
