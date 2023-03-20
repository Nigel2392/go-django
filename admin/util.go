package admin

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/admin/internal/menu"
	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/httputils/orderedmap"
	"github.com/Nigel2392/go-django/core/modelutils"
	"github.com/Nigel2392/go-django/core/modelutils/namer"

	"github.com/Nigel2392/router/v3/request"
	"golang.org/x/exp/slices"
)

// Generate data for the admin site's sidebar model list.
func groupModels(baseURL string, m []*models.Model, r *request.Request) []*menu.Item {
	var menuItemMap = make(map[string]*menu.Item)
	for _, model := range m {
		var user = r.User.(*auth.User)
		if !user.HasPerms(model.Permissions.View(), model.Permissions.List()) {
			continue
		}

		if mItem, ok := menuItemMap[model.AppName()]; !ok {
			mItem = menu.NewItem(
				httputils.TitleCaser.String(model.AppName()),
				fmt.Sprintf("%s/%s", baseURL, model.AppName()),
				fmt.Sprintf("/%s", model.AppName()),
				0,
				make([]*models.Model, 0),
			)
			mItem.Data = append(mItem.Data.([]*models.Model), model)
			mItem.Children(
				menu.NewItem(
					httputils.TitleCaser.String(model.Name),
					model.URLS.GroupURL(),
					fmt.Sprintf("/%s", model.URLS.GroupPart),
					0,
				),
			)
			menuItemMap[model.AppName()] = mItem
		} else {
			mItem.Children(
				menu.NewItem(
					httputils.TitleCaser.String(model.Name),
					model.URLS.GroupURL(),
					fmt.Sprintf("/%s", model.URLS.GroupPart),
					0,
				),
			)
		}
	}

	var apps = make([]*menu.Item, 0)
	for _, app := range menuItemMap {
		slices.SortFunc(app.Data.([]*models.Model), func(i, j *models.Model) bool {
			return strings.ToLower(i.Name) < strings.ToLower(j.Name)
		})
		apps = append(apps, app)
	}

	slices.SortFunc(apps, func(i, j *menu.Item) bool {
		if i.Weight == j.Weight {
			return i.Name < j.Name
		}
		return i.Weight < j.Weight
	})

	return apps
}

// Get table data from a list of models.
// This function is used to generate the table data for the list view.
func (as *AdminSite) tableDataFromModel(s any, d []any) [][]any {
	var data = make([][]any, 1)
	var valueOf = reflect.ValueOf(s)
	if valueOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
	}
	var firstRow = make([]any, 0)
	for i := 0; i < valueOf.NumField(); i++ {
		// Setup labels for the table
		var field = valueOf.Type().Field(i)
		var fieldType = field.Type

		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		var tags = field.Tag.Get("admin")
		if strings.Contains(tags, "hidden") {
			continue
		}
		if tags == "-" {
			continue
		}

		// Check if field is gorm.Model
		if modelutils.IsModelField(fieldType) {
			// Append the model fields to the first row
			firstRow = append(firstRow, "ID")
			firstRow = append(firstRow, "Created At")
			firstRow = append(firstRow, "Updated At")
			continue
		} else if fieldType.Kind() == reflect.Struct {
			firstRow = append(firstRow, field.Name)
			// firstRow = append(firstRow, fmt.Sprintf("%s ID", field.Name))
			continue
		} else if fieldType.Kind() == reflect.Slice {
			// CANNOT GET PRELOAD TO WORK!!! -> models.go/Model.Models() []any
			// CANNOT GET PRELOAD TO WORK!!! -> models.go/Model.Models() []any
			// CANNOT GET PRELOAD TO WORK!!! -> models.go/Model.Models() []any
			if !modelutils.IsModel(fieldType.Elem()) {
				firstRow = append(firstRow, fmt.Sprintf("%s Amount", field.Name))
			}
			continue
		}
		firstRow = append(firstRow, field.Name)
	}
	data[0] = firstRow

	for _, model := range d {
		var modelValue = reflect.ValueOf(model)
		if modelValue.Kind() == reflect.Ptr {
			modelValue = modelValue.Elem()
		}
		var row = make([]any, 0)
		for i := 0; i < modelValue.NumField(); i++ {
			var fieldValue = modelValue.Field(i)

			var field = modelValue.Type().Field(i)
			var tags = field.Tag.Get("admin")
			if tags == "-" {
				continue
			}
			var protectedTag = strings.Contains(tags, "protected")
			if strings.Contains(tags, "hidden") {
				continue
			}
			var fieldType = field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
				fieldValue = fieldValue.Elem()
			}

			if modelutils.IsModelField(fieldType) {
				var id = modelutils.GetID(fieldValue, "ID")
				var url, _, _ = as.getAdminDetailURL(model, id)
				var CreatedAt = fieldValue.FieldByName("CreatedAt").Interface()
				var UpdatedAt = fieldValue.FieldByName("UpdatedAt").Interface()

				if CreatedAt != nil {
					CreatedAt = formatTime(CreatedAt)
				}
				if UpdatedAt != nil {
					UpdatedAt = formatTime(UpdatedAt)
				}

				if protectedTag {
					row = append(row, template.HTML(fmt.Sprintf(`<a href="%s" class="table-link">%v</a>`, url, id)))
					row = append(row, "***************")
					row = append(row, "***************")
					continue
				}

				row = append(row, template.HTML(fmt.Sprintf(`<a href="%s" class="table-link">%v</a>`, url, id)))
				row = append(row, CreatedAt)
				row = append(row, UpdatedAt)
				continue
			} else if fieldType.Kind() == reflect.Struct {
				if !fieldValue.IsValid() {
					row = append(row, "")
					continue
				}
				row = append(row, modelutils.GetModelDisplay(fieldValue.Interface()))
				if protectedTag {
					row = append(row, "***************")
					continue
				}
				continue
			} else if fieldType.Kind() == reflect.Slice {

				// CANNOT GET PRELOAD TO WORK!!! -> models.go/Model.Models() []any
				// CANNOT GET PRELOAD TO WORK!!! -> models.go/Model.Models() []any
				// CANNOT GET PRELOAD TO WORK!!! -> models.go/Model.Models() []any
				if !modelutils.IsModel(fieldType.Elem()) {
					row = append(row, fieldValue.Len())
				}
				continue
			}
			if protectedTag {
				row = append(row, "***************")
				continue
			}
			row = append(row, fieldValue.Interface())
		}
		data = append(data, row)
	}

	return data
}

// Get a model's detail URL.
func (as *AdminSite) getAdminDetailURL(m any, id interface{}) (string, string, string) {
	var model = as.getModel(m)
	if model == nil {
		return "", "", ""
	}
	var url = model.URLS.DetailURL(id)
	return url, model.AppName(), model.Name
}

// Get a model from the admin site models.
func (as *AdminSite) getModel(m any) *models.Model {
	var name = namer.GetModelName(m)
	for _, model := range as.models {
		if name == model.Name {
			return model
		}
	}
	return nil
}

// Check if a user has permissions.
// If not, log the action.
func (as *AdminSite) checkPermission(user request.User, perms ...*auth.Permission) bool {
	var u, ok = user.(*auth.User)
	if !ok {
		return true
	}
	var hasPerm = u.HasPerms(perms...)
	if !hasPerm {
		var permList = make([]string, 0)
		for _, perm := range perms {
			permList = append(permList, perm.String())
		}
		as.Logger.Debugf("User %s does not have permission. Metadata: %v\n", u.Username, permList)
		var log = SimpleLog(u, LogActionUnauthorized)
		log.Meta.Set("permissions", permList)
		log.Save(as)
	}
	return hasPerm
}

// Add  default data to a request.
// (Title, apps, internals)
func defaultDataFunc(as *AdminSite, r *request.Request, title ...string) {
	__defaultDataFunc(as, r, title...)
}

// Add  default data to a request.
// (Title, apps, internals)
func __defaultDataFunc(as *AdminSite, r *request.Request, title ...string) {
	var t = as.Name
	if len(title) > 0 {
		t = title[0]
	}
	r.Data.Set("title", t)
	r.Data.Set("apps", as.sortedApps(r))
}

// Join a list of types into a string.
func joinFunc(args ...any) string {
	var s = make([]string, len(args))
	for i, arg := range args {
		s[i] = formatFunc(arg)
	}
	return strings.Join(s, "")
}

// Format a type.
// If the type can not be formatted, fmt.Sprint.
func formatFunc(a any) string {
	if a == nil {
		return ""
	}
	if modelutils.IsModel(a) {
		return modelutils.GetModelDisplay(a)
	}
	switch a := a.(type) {
	case time.Time:
		return a.Format("15:04:05 02-01-2006")
	}
	var t = reflect.TypeOf(a)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		var s = reflect.ValueOf(a)
		var l = s.Len()
		var r = make([]string, l)
		for i := 0; i < l; i++ {
			r[i] = fmt.Sprint(s.Index(i).Interface())
		}
		return strings.Join(r, ", ")
	case reflect.Map:
		var s = reflect.ValueOf(a)
		var l = s.Len()
		var r = make([]string, l)
		var i = 0
		for _, k := range s.MapKeys() {
			r[i] = k.String() + ": " + fmt.Sprint(s.MapIndex(k).Interface())
			i++
		}
		return strings.Join(r, ", ")
	}
	return fmt.Sprint(a)
}

// Format a time.Time or *time.Time to a string.
// This is used to format strings accordingly.
func formatTime(t any) string {
	switch t := t.(type) {
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	case *time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		return ""
	}
}

// Cut a string or []byte to a maximum length.
// If the string is longer than max, it will be cut and "..." will be appended.
func maxStrLenFunc(s any, max int) any {
	switch v := s.(type) {
	case string:
		if len(v) > max {
			return v[:max] + "..."
		}
		return v
	case []byte:
		if len(v) > max {
			return append(v[:max], []byte("...")...)
		}
		return v
	default:
		return s
	}
}

// Divide two numbers.
func divideFunc(a, b int) int {
	if b == 0 || a == 0 {
		return 0
	}
	return a / b
}

// Retrieve the *AdminURLs URLs for a model.
func (as *AdminSite) GetAdminURLs(m any) *models.AdminURLs {
	var model = as.getModel(m)
	if model == nil {
		return nil
	}
	return model.URLS
}

func (as *AdminSite) sortedApps(r *request.Request) []*menu.Item {
	var appMap = orderedmap.New[string, *menu.Item]()
	if r.User == nil {
		return []*menu.Item{}
	}
	var user = r.User.(*auth.User)
	if !user.IsAuthenticated() {
		return []*menu.Item{}
	}
	if user.HasPerms(PermissionViewAdminInternal) {
		var internal_app_menu = as.internal_menu_items()
		if len(internal_app_menu.Children()) > 0 {
			appMap.Set(internal_app_menu.Name, internal_app_menu)
		}
	}
	if user.HasPerms(PermissionViewAdminExtensions) {
		var extension_app_menu = as.extensions_menu_items()
		if len(extension_app_menu.Children()) > 0 {
			appMap.Set(extension_app_menu.Name, extension_app_menu)
		}
	}

	//	var newMenuItem = menu.NewItem("Apps", "", "", 0)
	//	newMenuItem.Children(groupModels(adminSite_models, r)...)
	//	appMap.Set(newMenuItem.Name, newMenuItem)
	for _, app := range groupModels(as.URL, as.models, r) {
		appMap.Set(app.Name, app)
	}

	//	if len(r.Request.URL.Query()["app_order"]) > 0 {
	//		return appMap.SortBySlice(
	//			r.Request.URL.Query()["app_order"]...,
	//		)
	//	}

	return appMap.SortBySlice(
		as.AppOrder...,
	)
}

//
//	func orderMap[T1 comparable, T2 any](k []T1, m map[T1]T2, condition func(t T2) bool) []T2 {
//		var result = make([]T2, len(k))
//		for i, key := range k {
//			var t = m[key]
//			if condition(t) {
//				result[i] = t
//			}
//		}
//		return result
//	}
