package openauth2models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// These fields are synonymous with the field names in  the database table
// Any update to the database table should be reflected here
const (
	FieldID               = "id"
	FieldUniqueIdentifier = "unique_identifier"
	FieldData             = "data"
	FieldCreatedAt        = "created_at"
	FieldUpdatedAt        = "updated_at"
	FieldIsAdministrator  = "is_administrator"
	FieldIsActive         = "is_active"
)

var ValidFields = []string{
	FieldID,
	FieldUniqueIdentifier,
	FieldData,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldIsAdministrator,
	FieldIsActive,
}

var _validFields = map[string]struct{}{
	FieldID:               {},
	FieldUniqueIdentifier: {},
	FieldData:             {},
	FieldCreatedAt:        {},
	FieldUpdatedAt:        {},
	FieldIsAdministrator:  {},
	FieldIsActive:         {},
}

func IsValidField(fieldName string) bool {
	var _, ok = _validFields[fieldName]
	return ok
}

type User struct {
	ID               uint64          `json:"id"`
	UniqueIdentifier string          `json:"unique_identifier"`
	ProviderName     string          `json:"provider_name"`
	Data             json.RawMessage `json:"data"`
	AccessToken      string          `json:"access_token"`
	RefreshToken     string          `json:"refresh_token"`
	TokenType        string          `json:"token_type"`
	ExpiresAt        time.Time       `json:"expires_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	IsAdministrator  bool            `json:"is_administrator"`
	IsActive         bool            `json:"is_active"`

	IsLoggedIn bool `json:"is_logged_in"`
	context    context.Context
}

func (u *User) String() string {
	return fmt.Sprintf(
		"%s (%s)",
		u.UniqueIdentifier,
		u.ProviderName,
	)
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn
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
	u.AccessToken = token.AccessToken
	u.RefreshToken = token.RefreshToken
	u.ExpiresAt = token.Expiry
	u.TokenType = token.TokenType
}

func (u *User) Token() *oauth2.Token {
	var token = &oauth2.Token{
		AccessToken:  u.AccessToken,
		RefreshToken: u.RefreshToken,
		Expiry:       u.ExpiresAt,
		TokenType:    u.TokenType,
	}
	return token
}
