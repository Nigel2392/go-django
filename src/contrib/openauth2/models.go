package openauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"golang.org/x/oauth2"
)

// A UserBinder is an interface that allows binding a User to a content object.
//
// It is used to bind the User to the content object after it has been scanned
// from the database. This is useful for content objects that need to know
// about the User that owns them.
type UserBinder interface {
	BindUser(user *User) error
}

// A UserFormDefiner is an interface that allows a content object to define
// form fields for a User.
type UserFormDefiner interface {
	// UserFormFields returns the form fields for the User.
	//
	// It is used to define the form fields for the User in the admin interface and
	// other places where forms might be used.
	UserFormFields(user *User) []attrs.Field
}

var (
	_ attrs.Definer                 = &User{}
	_ queries.ActsBeforeSave        = &User{}
	_ queries.ActsBeforeCreate      = &User{}
	_ queries.ActsAfterCreate       = &User{}
	_ queries.ActsBeforeUpdate      = &User{}
	_ queries.ActsAfterUpdate       = &User{}
	_ queries.ActsBeforeDelete      = &User{}
	_ queries.ActsAfterDelete       = &User{}
	_ queries.UniqueTogetherDefiner = &User{}
)

type User struct {
	models.Model `table:"openauth2_users"`
	users.Base
	ID               uint64            `json:"id"`
	UniqueIdentifier string            `json:"unique_identifier"`
	ProviderName     string            `json:"provider_name"`
	Data             json.RawMessage   `json:"data"`
	AccessToken      drivers.Text      `json:"access_token"`
	RefreshToken     drivers.Text      `json:"refresh_token"`
	TokenType        string            `json:"token_type"`
	ExpiresAt        drivers.Timestamp `json:"expires_at"`
	CreatedAt        drivers.DateTime  `json:"created_at"`
	UpdatedAt        drivers.DateTime  `json:"updated_at"`

	context context.Context
	object  interface{}
}

func (u *User) String() string {
	return fmt.Sprintf(
		"%s (%s)",
		u.UniqueIdentifier,
		u.ProviderName,
	)
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator && u.IsActive
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn && u.IsActive
}

func (u *User) SetContext(ctx context.Context) *User {
	u.context = ctx
	return u
}

func (u *User) Context() context.Context {
	if u.context == nil {
		u.context = context.Background()
	}
	return u.context
}

func (u *User) SetToken(token *oauth2.Token) {
	u.AccessToken = drivers.Text(token.AccessToken)
	u.RefreshToken = drivers.Text(token.RefreshToken)
	u.ExpiresAt = drivers.Timestamp(token.Expiry)
	u.TokenType = token.TokenType
}

func (u *User) Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  string(u.AccessToken),
		RefreshToken: string(u.RefreshToken),
		Expiry:       u.ExpiresAt.Time(),
		TokenType:    u.TokenType,
	}
}

// Provider returns the AuthConfig for the User's provider.
func (u *User) Provider() (*AuthConfig, error) {
	var c, err = App.Provider(u.ProviderName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get provider %s: %w",
			u.ProviderName, err,
		)
	}
	return c, nil
}

func (u *User) ContentObject() (interface{}, error) {
	if u.object != nil {
		return u.object, nil
	}

	if len(u.Data) == 0 {
		return nil, errors.New("User has no data for it's content object")
	}

	var c, err = u.Provider()
	if err != nil {
		return nil, err
	}

	obj, err := c.ScanContentObject(
		bytes.NewReader(u.Data),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to scan content object for user %s: %w",
			u.UniqueIdentifier, err,
		)
	}

	if binder, ok := obj.(UserBinder); ok {
		if err := binder.BindUser(u); err != nil {
			return nil, fmt.Errorf(
				"failed to bind user %s to content object: %w",
				u.UniqueIdentifier, err,
			)
		}
	}

	u.object = obj
	return u.object, nil
}

func (o *User) BeforeCreate(ctx context.Context) error {
	if o.CreatedAt.Time().IsZero() {
		o.CreatedAt = drivers.CurrentDateTime()
	}

	return core.SIGNAL_BEFORE_USER_CREATE.Send(o)
}

func (o *User) AfterCreate(ctx context.Context) error {
	return core.SIGNAL_AFTER_USER_CREATE.Send(o)
}

func (o *User) BeforeUpdate(ctx context.Context) error {
	return core.SIGNAL_BEFORE_USER_UPDATE.Send(o)
}

func (o *User) AfterUpdate(ctx context.Context) error {
	return core.SIGNAL_AFTER_USER_UPDATE.Send(o)
}

func (o *User) BeforeDelete(ctx context.Context) error {
	return core.SIGNAL_BEFORE_USER_DELETE.Send(o.ID)
}

func (o *User) AfterDelete(ctx context.Context) error {
	return core.SIGNAL_AFTER_USER_DELETE.Send(o.ID)
}

func (o *User) BeforeSave(ctx context.Context) error {
	o.UpdatedAt = drivers.CurrentDateTime()
	return nil
}

func (o *User) UniqueTogether() [][]string {
	return [][]string{
		{"UniqueIdentifier", "ProviderName"},
	}
}

func (o *User) FieldDefs() attrs.Definitions {
	return o.Model.Define(o,
		o.Fields,
		o.contentObjectFields,
	)
}

func (o *User) Fields(this attrs.Definer) []any {
	return []any{
		attrs.NewField(
			o, "ID", &attrs.FieldConfig{
				Null:     false,
				Blank:    false,
				ReadOnly: true,
				Label:    "ID",
				Primary:  true,
				Column:   "id",
			},
		),
		attrs.NewField(
			o, "UniqueIdentifier", &attrs.FieldConfig{
				Null:      false,
				Blank:     false,
				ReadOnly:  true,
				Label:     "Unique Identifier",
				Column:    "unique_identifier",
				MaxLength: 255,
			},
		),
		attrs.NewField(
			o, "ProviderName", &attrs.FieldConfig{
				Null:      false,
				Blank:     false,
				ReadOnly:  true,
				Label:     "Provider Name",
				Column:    "provider_name",
				MaxLength: 255,
			},
		),
		attrs.NewField(
			o, "Data", &attrs.FieldConfig{
				Null:     false,
				Blank:    true,
				ReadOnly: true,
				Label:    "Data",
				Column:   "data",
			},
		),
		attrs.NewField(
			o, "AccessToken", &attrs.FieldConfig{
				Null:     false,
				Blank:    true,
				ReadOnly: true,
				Label:    "Access Token",
				Column:   "access_token",
			},
		),
		attrs.NewField(
			o, "RefreshToken", &attrs.FieldConfig{
				Null:     false,
				Blank:    true,
				ReadOnly: true,
				Label:    "Refresh Token",
				Column:   "refresh_token",
			},
		),
		attrs.NewField(
			o, "TokenType", &attrs.FieldConfig{
				Null:      false,
				Blank:     true,
				ReadOnly:  true,
				Label:     "Token Type",
				Column:    "token_type",
				MaxLength: 50,
			},
		),
		attrs.NewField(
			o, "ExpiresAt", &attrs.FieldConfig{
				Null:      false,
				Blank:     true,
				ReadOnly:  true,
				MaxLength: 6, // precision for fractional seconds
				Label:     "Expires At",
				Column:    "expires_at",
			},
		),
		attrs.NewField(
			o, "CreatedAt", &attrs.FieldConfig{
				Null:     false,
				Blank:    true,
				ReadOnly: true,
				Label:    "Created At",
				Column:   "created_at",
			},
		),
		attrs.NewField(
			o, "UpdatedAt", &attrs.FieldConfig{
				Null:     false,
				Blank:    true,
				ReadOnly: true,
				Label:    "Updated At",
				Column:   "updated_at",
			},
		),
		o.Base.Fields(this),
	}
}

func (o *User) contentObjectFields(this attrs.Definer) []attrs.Field {
	var fields = make([]attrs.Field, 0)
	if o.ProviderName != "" && o.UniqueIdentifier != "" && len(o.Data) > 0 {
		var provider, err = o.Provider()
		if err != nil {
			panic(fmt.Sprintf(
				"failed to get provider %s for user %s: %s",
				o.ProviderName, o.UniqueIdentifier, err,
			))
		}

		// pre check if it adheres to UserFormDefiner
		// to avoid unnecessary calls to ContentObject()
		if _, ok := provider.DataStruct.(UserFormDefiner); ok {
			var contentObject, err = o.ContentObject()
			if err != nil {
				panic(fmt.Sprintf(
					"failed to get content object for user %s: %s",
					o.UniqueIdentifier, err,
				))
			}

			var oDefiner, ok = contentObject.(UserFormDefiner)
			if !ok {
				panic(fmt.Sprintf(
					"content object for user %s does not implement UserFormDefiner: %T",
					o.UniqueIdentifier, contentObject,
				))
			}

			var contentFields = oDefiner.UserFormFields(o)
			fields = append(fields, contentFields...)
		}
	}
	return fields
}
