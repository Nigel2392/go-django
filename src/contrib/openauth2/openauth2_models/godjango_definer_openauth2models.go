package openauth2models

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func (o *User) FieldDefs() attrs.Definitions {
	var fields = make([]attrs.Field, 12)
	fields[0] = attrs.NewField(
		o, "ID", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "ID",
			Primary:  true,
			ReadOnly: true,
		},
	)
	fields[1] = attrs.NewField(
		o, "UniqueIdentifier", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Unique Identifier",
			ReadOnly: true,
		},
	)
	fields[2] = attrs.NewField(
		o, "ProviderName", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Provider Name",
			ReadOnly: true,
		},
	)
	fields[3] = attrs.NewField(
		o, "Data", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Data",
			ReadOnly: true,
		},
	)
	fields[4] = attrs.NewField(
		o, "AccessToken", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Access Token",
			ReadOnly: true,
		},
	)
	fields[5] = attrs.NewField(
		o, "RefreshToken", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Refresh Token",
			ReadOnly: true,
		},
	)
	fields[6] = attrs.NewField(
		o, "TokenType", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Token Type",
			ReadOnly: true,
		},
	)
	fields[7] = attrs.NewField(
		o, "ExpiresAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Expires At",
			ReadOnly: true,
		},
	)
	fields[8] = attrs.NewField(
		o, "CreatedAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Created At",
			ReadOnly: true,
		},
	)
	fields[9] = attrs.NewField(
		o, "UpdatedAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			Label:    "Updated At",
			ReadOnly: true,
		},
	)
	fields[10] = attrs.NewField(
		o, "IsAdministrator", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: "Is Administrator",
		},
	)
	fields[11] = attrs.NewField(
		o, "IsActive", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: "Is Active",
		},
	)
	return attrs.Define(o, fields...)
}
