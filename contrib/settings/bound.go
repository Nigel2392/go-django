package settings

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/forms"
)

var _ BoundSetting = (*boundSetting)(nil)

type BoundSettingGroup struct {
	ctx      context.Context
	name     string
	group    SettingsGroup
	Settings []*boundSetting
}

func (b BoundSettingGroup) Icon() template.HTML {
	if b.group.Icon != nil {
		return b.group.Icon(b.ctx)
	}
	return ""
}

func (b BoundSettingGroup) Label() string {
	if b.group.Label != nil {
		return b.group.Label(b.ctx)
	}
	return ""
}

func (b BoundSettingGroup) Description() string {
	if b.group.Description != nil {
		return b.group.Description(b.ctx)
	}
	return ""
}

type boundSetting struct {
	ctx        context.Context
	definition Setting
	instance   *RegisteredSetting
	value      any
}

func (b *boundSetting) Name() string {
	return b.instance.InternalName
}
func (b *boundSetting) DBType() dbtype.Type {
	return b.definition.DBType()
}
func (b *boundSetting) Label() string {
	return b.definition.Label(b.ctx)
}
func (b *boundSetting) Icon() template.HTML {
	return b.definition.Icon(b.ctx)
}
func (b *boundSetting) Description() string {
	return b.definition.Description(b.ctx)
}
func (b *boundSetting) FormField() forms.Field {
	return b.definition.FormField(b.ctx)
}
func (b *boundSetting) Permissions() []string {
	return b.definition.Permissions()
}
func (b *boundSetting) DefaultValue() any {
	return b.definition.DefaultValue()
}
func (b *boundSetting) Value() (val any, err error) {
	if b.value != nil {
		return b.value, nil
	}

	if b.instance.Data == nil {
		return nil, nil
	}

	val, err = b.definition.ValueFromDB(b.ctx, b.instance.Data)
	if err != nil {
		return nil, err
	}

	b.value = val
	return val, err
}
