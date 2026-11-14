package settings

import (
	"context"
	"encoding/json"
	"html/template"
	"iter"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/elliotchance/orderedmap/v2"
)

type requestContextKey struct{}

type Setting interface {
	DBType() dbtype.Type
	Permissions() []string
	Icon(ctx context.Context) template.HTML
	Label(ctx context.Context) string
	Description(ctx context.Context) string
	FormField(ctx context.Context) forms.Field
	DefaultValue() any
	ValueFromDB(context.Context, json.RawMessage) (any, error)
}

type BoundSetting interface {
	Icon() template.HTML
	Name() string
	Label() string
	Description() string
	Value() (any, error)
	DBType() dbtype.Type
	DefaultValue() any
	FormField() forms.Field
	Permissions() []string
}

func RegisterGroup(ordering int, groupName string, group SettingsGroup) {
	reg.registerGroup(ordering, groupName, group)
}

func RegisterSetting(ordering int, group, name string, setting Setting) {
	reg.registerSetting(ordering, group, name, setting)
}

func RequestWithSettings(r *http.Request) (*http.Request, error) {
	m, err := SettingsForRequest(r)
	if err != nil {
		return nil, err
	}

	r = r.WithContext(context.WithValue(
		r.Context(), requestContextKey{}, m,
	))

	return r, nil
}

func SettingsForRequest(r *http.Request, refresh ...bool) (m *orderedmap.OrderedMap[string, BoundSettingGroup], err error) {
	var ok bool
	if len(refresh) > 0 && refresh[0] {
		goto freshSettings
	}

	m, ok = r.Context().Value(requestContextKey{}).(*orderedmap.OrderedMap[string, BoundSettingGroup])
	if ok && m != nil {
		return m, nil
	}

freshSettings:
	groupIter, err := groupedSettings(r.Context())
	if err != nil {
		return nil, err
	}

	m = orderedmap.NewOrderedMap[string, BoundSettingGroup]()
	for bound := range groupIter {
		var newSettings = make([]*boundSetting, 0, len(bound.Settings))
		for _, setting := range bound.Settings {
			if perms := setting.definition.Permissions(); len(perms) > 0 && !permissions.HasPermission(r, perms...) {
				// no permissions for setting
				continue
			}

			newSettings = append(newSettings, setting)
		}

		bound.Settings = newSettings
		m.Set(bound.name, bound)
	}

	return m, nil
}

func groupedSettings(ctx context.Context) (iter.Seq[BoundSettingGroup], error) {

	var names = make([]string, 0)
	for _, s := range reg.groups {
		for _, setting := range s.settings {
			names = append(names, setting.internalName)
		}
	}

	var n, iter, err = queries.GetQuerySet(&RegisteredSetting{}).
		WithContext(ctx).
		Filter("InternalName__in", names).
		IterAll()
	if err != nil {
		return nil, err
	}

	var instances = make(map[string]*RegisteredSetting, n)
	for row, err := range iter {
		if err != nil {
			return nil, err
		}

		instances[row.Object.InternalName] = row.Object
	}

	return func(yield func(BoundSettingGroup) bool) {
		for _, regGroup := range reg.groups {
			var group = BoundSettingGroup{
				ctx:      ctx,
				name:     regGroup.name,
				group:    regGroup.SettingsGroup,
				Settings: make([]*boundSetting, 0, len(regGroup.settings)),
			}

			for _, setting := range regGroup.settings {
				instance, ok := instances[setting.internalName]
				if !ok {
					instance = new(RegisteredSetting{
						InternalName: setting.internalName,
					})
				}

				group.Settings = append(group.Settings, &boundSetting{
					ctx:        ctx,
					definition: setting.setting,
					instance:   instance,
				})
			}

			if !yield(group) {
				break
			}
		}
	}, nil
}
