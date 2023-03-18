package models

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/models"
	"github.com/Nigel2392/go-django/core/modelutils"
	"github.com/Nigel2392/go-django/core/modelutils/namer"
	"github.com/google/uuid"

	"gorm.io/gorm"
)

// This URL is not usable, and is used to generate the actual absolute URL of the model.
type AdminURLs struct {
	// The URL for the admin site.
	AdminSite string
	// AppName is the name of the app the model belongs to.
	AppName string
	// The URL for the list/group name view of the model.
	GroupPart string
	// The URL for the create view of the model.
	Create string
	// The URL for the detail view of the model.
	Detail string
	// The URL for the delete view of the model.
	Delete string
}

// The app url where the model is registered.
func (a *AdminURLs) AppURL() string {
	return httputils.NicePath(false, a.AdminSite, a.AppName)
}

// The group url where the model is registered.
func (a *AdminURLs) GroupURL() string {
	return httputils.NicePath(false, a.AppURL(), a.GroupPart)
}

// The detail url where you can view/edit the model.
func (a *AdminURLs) DetailURL(id any) string {
	return fmt.Sprintf("%s/%v", a.GroupURL(), id)
}

// The delete url where you can delete the model.
func (a *AdminURLs) DeleteURL(id any) string {
	return fmt.Sprintf("%s/delete", a.DetailURL(id))
}

// The create url where you can create an instance of the model.
func (a *AdminURLs) CreateURL() string {
	return fmt.Sprintf("%s/%s", a.GroupURL(), a.Create)
}

type Model struct {
	// The name of the model.
	Name string
	// The URL for the model.
	URLS *AdminURLs
	// The model itself.
	Mdl any
	// List of fields to display in the admin.
	// Fields []*Field
	Permissions auth.PermissionMap
}

func NewModel(base_url string, m any, permsDB *gorm.DB) (*Model, error) {
	var name = namer.GetModelName(m)
	var model = &Model{
		Name: name,
		URLS: &AdminURLs{
			AdminSite: base_url,
			AppName:   httputils.SimpleSlugify(namer.GetAppName(m)),
			GroupPart: fmt.Sprintf("/%s", httputils.SimpleSlugify(name)),
			Create:    "/create",
		},
		Mdl:         m,
		Permissions: auth.NewPermissionMap(m),
	}
	var fieldVal, err = modelutils.GetField(m, "ID")
	if err != nil {
		model.URLS.Detail = "/<<id:alphanum>>"
		model.URLS.Delete = "/<<id:alphanum>>/delete"
		goto next
	}

	switch fieldVal.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
		models.Model[int], models.Model[int8], models.Model[int16], models.Model[int32], models.Model[int64],
		models.Model[uint], models.Model[uint8], models.Model[uint16], models.Model[uint32], models.Model[uint64]:
		model.URLS.Detail = "/<<id:int>>"
		model.URLS.Delete = "/<<id:int>>/delete"
		goto next
	case models.Model[models.DefaultIDField],
		models.Model[uuid.UUID],
		models.DefaultIDField,
		uuid.UUID:
		model.URLS.Detail = "/<<id:uuid>>"
		model.URLS.Delete = "/<<id:uuid>>/delete"
		goto next
	default:
		model.URLS.Detail = "/<<id:alphanum>>"
		model.URLS.Delete = "/<<id:alphanum>>/delete"
		goto next
	}
next:
	var perms = model.Permissions.All()
	// Save the permissions to the database.
	for i, perm := range perms {
		var err = permsDB.FirstOrCreate(&perms[i], perm).Error
		if err != nil {
			return nil, err
		}
	}

	return model, nil
}

func (m *Model) AppName() string {
	return m.URLS.AppName
}

func (m *Model) URL() string {
	return fmt.Sprintf("%s/%s/%s", m.URLS.AdminSite, m.URLS.AppName, m.Name)
}

func (m *Model) New() any {
	return modelutils.GetNewModel(m.Mdl, true)
}

func (m *Model) NewSlice() any {
	return modelutils.GetNewModelSlice(m.Mdl)
}

func (m *Model) Models(db *gorm.DB, editQuery ...func(tx *gorm.DB) *gorm.DB) []any {
	return m.models(db, editQuery...)
}

func (m *Model) models(db *gorm.DB, editQuery ...func(tx *gorm.DB) *gorm.DB) []any {
	var modelSlice = modelutils.GetNewModelSlice(m.Mdl)
	db = db.Model(m.Mdl)
	if len(editQuery) > 0 {
		for _, qf := range editQuery {
			db = qf(db)
		}
	}
	var preloadFields, joinFields = modelutils.GetPreloadFields(m.Mdl)
	for _, field := range preloadFields {
		db = db.Preload(field)
	}
	for _, field := range joinFields {
		db = db.Joins(field)
	}
	db.Find(&modelSlice)

	var modelValue = reflect.ValueOf(modelSlice)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}
	var models = make([]any, modelValue.Len())
	for i := 0; i < modelValue.Len(); i++ {
		models[i] = modelValue.Index(i).Interface()
	}
	return models
}
