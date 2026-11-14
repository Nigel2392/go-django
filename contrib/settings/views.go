package settings

import (
	"encoding/json"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/messages"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/versatile"
)

func viewSettings(w http.ResponseWriter, r *http.Request) {

	if !permissions.HasPermission(r, "settings:view") {
		autherrors.Fail(
			http.StatusInternalServerError,
			"user has no permissions",
		)
	}

	var groupsMap, err = SettingsForRequest(r)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to retrieve groups: %v", err,
		)
	}

	var formFields = make([]forms.Field, 0)
	var initial = make(map[string]interface{}, groupsMap.Len())
	for head := groupsMap.Front(); head != nil; head = head.Next() {

		block := blocks.NewStructBlock()
		initialBlock := new(blocks.StructBlockValue{
			BlockObject: block,
			V:           make(map[string]interface{}),
		})

		for _, setting := range head.Value.Settings {

			blockOpts := make([]func(*blocks.FieldBlock), 0)
			blockOpts = append(blockOpts,
				blocks.WithLabel[*blocks.FieldBlock](setting.Label()),
			)

			if desc := setting.Description(); desc != "" {
				blockOpts = append(blockOpts,
					blocks.WithHelpText[*blocks.FieldBlock](desc))
			}

			fieldBlock := blocks.FormFieldBlock(
				setting.FormField(),
				drivers.DBToDefaultGoType(setting.DBType()),
				blockOpts...,
			)
			block.AddField(setting.Name(), fieldBlock)

			value, err := setting.Value()
			if err != nil || versatile.IsZero(value) {
				value = setting.DefaultValue()
			}

			initialBlock.V[setting.instance.InternalName] = &blocks.FieldBlockValue{
				BlockObject: fieldBlock,
				V:           value,
			}
		}

		opts := make([]func(fields.Field), 0)
		opts = append(opts,
			fields.Label(head.Value.Label()),
			fields.Name(head.Value.name),
			fields.ReadOnly(true),
		)

		if desc := head.Value.Description(); desc != "" {
			opts = append(opts,
				fields.HelpText(desc))
		}

		initial[head.Value.name] = initialBlock

		formFields = append(
			formFields,
			blocks.BlockField(block, opts...),
		)
	}

	var form = forms.NewBaseForm(r.Context())
	for _, field := range formFields {
		field.SetReadOnly(true)
		form.AddField(field.Name(), field)
	}

	form.Defaults = initial

	context := admin.NewContext(r, admin.AdminSite, nil)
	context.Set("form", form)
	context.SetPage(admin.PageOptions{
		Request: r,
		TitleFn: trans.S("Site Settings"),
		HeaderActions: []components.ShowableComponent{
			components.NewShowableComponent(r, func(r *http.Request) bool { return true }, components.Link(
				components.ButtonConfig{
					Text: trans.S("Change"),
					Type: components.ClassTypeSecondary,
				},
				func() string { return django.Reverse("admin:settings:edit") },
			)),
		},
	})

	err = tpl.FRender(
		w, context,
		"admin", "settings/detail.tmpl",
	)
	if err != nil {
		panic(err)
	}
}

func viewSettingsEdit(w http.ResponseWriter, r *http.Request) {

	if !permissions.HasPermission(r, "settings:edit") {
		autherrors.Fail(
			http.StatusInternalServerError,
			"user has no permissions",
		)
	}

	var groupsMap, err = SettingsForRequest(r)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to retrieve groups: %v", err,
		)
	}

	var formFields = make([]forms.Field, 0)
	var groups = make([]BoundSettingGroup, 0, groupsMap.Len())
	var initial = make(map[string]interface{}, groupsMap.Len())
	for head := groupsMap.Front(); head != nil; head = head.Next() {

		block := blocks.NewStructBlock()
		initialBlock := new(blocks.StructBlockValue{
			BlockObject: block,
			V:           make(map[string]interface{}),
		})
		for _, setting := range head.Value.Settings {

			blockOpts := make([]func(*blocks.FieldBlock), 0)
			blockOpts = append(blockOpts,
				blocks.WithLabel[*blocks.FieldBlock](setting.Label()))

			if desc := setting.Description(); desc != "" {
				blockOpts = append(blockOpts,
					blocks.WithHelpText[*blocks.FieldBlock](desc))
			}

			fieldBlock := blocks.FormFieldBlock(
				setting.FormField(),
				drivers.DBToDefaultGoType(setting.DBType()),
				blockOpts...,
			)
			block.AddField(setting.Name(), fieldBlock)

			value, err := setting.Value()
			if err != nil {
				except.Fail(
					http.StatusInternalServerError,
					"failed to retrieve value for setting %q: %v", setting.Name(), err,
				)
			}

			if versatile.IsZero(value) {
				value = setting.DefaultValue()
			}

			initialBlock.V[setting.instance.InternalName] = &blocks.FieldBlockValue{
				BlockObject: fieldBlock,
				V:           value,
			}
		}

		opts := make([]func(fields.Field), 0)
		opts = append(opts,
			fields.Label(head.Value.Label()),
			fields.Name(head.Value.name),
		)

		if desc := head.Value.Description(); desc != "" {
			opts = append(opts,
				fields.HelpText(desc))
		}

		initial[head.Value.name] = initialBlock
		groups = append(groups, head.Value)
		formFields = append(
			formFields,
			blocks.BlockField(block, opts...),
		)
	}

	var form = forms.NewBaseForm(r.Context())
	for _, field := range formFields {
		form.AddField(field.Name(), field)
	}

	form.SetInitial(initial)

	if r.Method == http.MethodPost {

		form = forms.Initialize(form,
			forms.WithRequestData(http.MethodPost, r),
		)

		if forms.IsValid(r.Context(), form) {
			settings := make([]*RegisteredSetting, 0)
			cleaned := form.CleanedData()
			for _, group := range groups {
				data, ok := cleaned[group.name]
				if !ok {
					// logger.Debugf("%q is not found in cleaned data %v", group.name, slices.Collect(maps.Keys(cleaned)))
					continue
				}

				field, ok := form.Field(group.name)
				if !ok {
					// logger.Debugf("%q is not found in form fields %v", group.name, form.FieldMap().Keys())
					continue
				}

				if !field.HasChanged(initial[group.name], data) {
					logger.Debugf("%q has not changed", group.name)
					continue
				}

				sb, ok := data.(*blocks.StructBlockValue)
				if !ok {
					// logger.Debugf("%q is not a *StructBlockValue: %T", group.name, data)
					continue
				}

				errs := blocks.NewBlockErrors[string]()
				for _, setting := range group.Settings {
					val, ok := sb.V[setting.instance.InternalName]
					if !ok {
						// logger.Debugf("%q is not found in cleaned data %v", setting.instance.InternalName, slices.Collect(maps.Keys(sb.V)))
						continue
					}

					data, err := json.Marshal(val)
					if err != nil {
						errs.AddError(setting.instance.InternalName, err)
						continue
					}

					settings = append(settings, &RegisteredSetting{
						ID:           setting.instance.ID,
						InternalName: setting.instance.InternalName,
						Data:         data,
					})
				}

				if errs.Len() > 0 {
					form.AddError(group.name, errs)
					continue
				}
			}

			if !forms.HasErrors(form) {
				if len(settings) == 0 {
					messages.Warning(r, "No settings saved.")
					http.Redirect(w, r, django.Reverse("admin:settings"), http.StatusSeeOther)
					return
				}

				_, err := queries.
					GetQuerySet(&RegisteredSetting{}).
					Select("Data").
					BulkUpdate(settings)
				if err == nil {
					messages.Success(r, "Settings saved succesfully.")
					http.Redirect(w, r, django.Reverse("admin:settings"), http.StatusSeeOther)
					return
				}

				form.AddFormError(err)
			}
		}
	}

	context := admin.NewContext(r, admin.AdminSite, nil)
	context.Set("groupedSettings", groups)
	context.Set("form", form)
	context.SetPage(admin.PageOptions{
		Request: r,
		TitleFn: trans.S("Change Site Settings"),
	})

	err = tpl.FRender(
		w, context,
		"admin", "settings/change.tmpl",
	)
	if err != nil {
		panic(err)
	}
}
