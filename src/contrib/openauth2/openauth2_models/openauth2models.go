package openauth2models

import (
	"encoding/json"
	"time"
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
	ExpiresAt        time.Time       `json:"expires_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	IsAdministrator  bool            `json:"is_administrator"`
	IsActive         bool            `json:"is_active"`
	IsLoggedIn       bool            `json:"is_logged_in"`
}

func (u *User) String() string {
	return u.UniqueIdentifier
}

func (u *User) IsAdmin() bool {
	return u.IsAdministrator
}

func (u *User) IsAuthenticated() bool {
	return u.IsLoggedIn
}
