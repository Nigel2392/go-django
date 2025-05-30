package openauth2models

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var (
	_ widgets.Widget
)

func (o *User) FieldDefs() attrs.Definitions {
	var fields = make([]attrs.Field, 12)
	fields[0] = attrs.NewField(
		o, "ID", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "ID",
			Primary:  true,
		},
	)
	fields[1] = attrs.NewField(
		o, "UniqueIdentifier", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Unique Identifier",
		},
	)
	fields[2] = attrs.NewField(
		o, "ProviderName", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Provider Name",
		},
	)
	fields[3] = attrs.NewField(
		o, "Data", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Data",
		},
	)
	fields[4] = attrs.NewField(
		o, "AccessToken", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Access Token",
		},
	)
	fields[5] = attrs.NewField(
		o, "RefreshToken", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Refresh Token",
		},
	)
	fields[6] = attrs.NewField(
		o, "TokenType", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Token Type",
		},
	)
	fields[7] = attrs.NewField(
		o, "ExpiresAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Expires At",
		},
	)
	fields[8] = attrs.NewField(
		o, "CreatedAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Created At",
		},
	)
	fields[9] = attrs.NewField(
		o, "UpdatedAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Updated At",
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
