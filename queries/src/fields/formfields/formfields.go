package formfields

import (
	"context"
	"embed"
	"io/fs"
	"reflect"

	"github.com/Nigel2392/go-django/internal/django_reflect"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms"
	django_formfields "github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
	"github.com/Nigel2392/goldcrest"
)

type BaseRelationField struct {
	*django_formfields.BaseField
	Field    attrs.FieldDefinition
	Relation attrs.Relation
}

//go:embed assets/**
var assetsFS embed.FS

func init() {
	goldcrest.Register(forms.FormTemplateStaticHookName, 0, forms.FormTemplateStaticHook(func(fSys fs.FS) fs.FS {
		if mfs, ok := fSys.(*filesystem.MultiFS); ok {
			mfs.Add(filesystem.Sub(assetsFS, "assets/static"), nil)
			return mfs
		}

		fSys = filesystem.NewMultiFS(
			fSys,
			filesystem.Sub(assetsFS, "assets/static"),
		)

		return fSys
	}))

	goldcrest.Register(forms.FormTemplateFSHookName, 0, forms.FormTemplateFSHook(func(fSys fs.FS, cnf *tpl.Config) fs.FS {
		if mfs, ok := fSys.(*filesystem.MultiFS); ok {
			mfs.Add(filesystem.Sub(assetsFS, "assets/templates"), nil)
			return mfs
		}

		fSys = filesystem.NewMultiFS(
			fSys,
			filesystem.Sub(assetsFS, "assets/templates"),
		)

		return fSys
	}))

	goldcrest.Register(attrs.HookFormFieldForType, 100, // order 100 to make sure other more specific hooks get a chance to run before this one.
		attrs.FormFieldGetter(func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(django_formfields.Field)) (django_formfields.Field, bool) {
			var rel = f.Rel()
			if rel == nil {
				return nil, false
			}

			var relType = rel.Type()
			switch relType {
			case attrs.RelManyToOne:
				return &ForeignKeyFormField{
					BaseRelationField: BaseRelationField{
						BaseField: django_formfields.NewField(opts...),
						Field:     f,
						Relation:  rel,
					},
				}, true
			case attrs.RelManyToMany:
				return &ManyToManyFormField{
					BaseRelationField: BaseRelationField{
						BaseField: django_formfields.NewField(opts...),
						Field:     f,
						Relation:  rel,
					},
				}, true
			case attrs.RelOneToOne:
				return &OneToOneFormField{
					ForeignKeyFormField: ForeignKeyFormField{
						BaseRelationField: BaseRelationField{
							BaseField: django_formfields.NewField(opts...),
							Field:     f,
							Relation:  rel,
						},
					},
				}, true
			case attrs.RelOneToMany:
			default:
				assert.Fail("unknown relation type %s for field %s", relType, f.Name())
			}
			return nil, false
		}),
	)
}

type ForeignKeyFormField struct {
	BaseRelationField
}

func (f *ForeignKeyFormField) HasChanged(initial, data interface{}) bool {
	if initial == nil && data == nil {
		return false
	}

	var (
		a, ok1 = initial.(attrs.Definer)
		b, ok2 = data.(attrs.Definer)
	)

	if ok1 && ok2 && (!django_reflect.IsZero(a) && !django_reflect.IsZero(b)) {
		var pkA = attrs.PrimaryKey(context.Background(), a)
		var pkB = attrs.PrimaryKey(context.Background(), b)
		return f.BaseField.HasChanged(pkA, pkB)
	}

	if ok1 && !ok2 && !django_reflect.IsZero(a) {
		var pkA = attrs.PrimaryKey(context.Background(), a)
		return f.BaseField.HasChanged(pkA, data)
	}

	if !ok1 && ok2 && !django_reflect.IsZero(b) {
		var pkB = attrs.PrimaryKey(context.Background(), b)
		return f.BaseField.HasChanged(initial, pkB)
	}

	return f.BaseField.HasChanged(initial, data)
}

func (f *ForeignKeyFormField) ValueToForm(value interface{}) interface{} {
	//	if value == nil {
	//		return nil
	//	}
	//
	//	if attrs.IsZero(value) {
	//		return nil
	//	}
	//
	//	switch v := value.(type) {
	//	case attrs.Definer:
	//		var defs = v.FieldDefs()
	//		var prim = defs.Primary()
	//		return prim.GetValue()
	//	default:
	//		return value
	//	}
	return f.Widget().ValueToForm(value)
}

func (f *ForeignKeyFormField) ValueToGo(value interface{}) (interface{}, error) {

	//	if _, ok := value.(attrs.Definer); ok {
	//		return value, nil
	//	}
	//
	//	var newObj = attrs.NewObject[attrs.Definer](f.Field.Rel().Model())
	//	var defs = newObj.FieldDefs()
	//	var prim = defs.Primary()
	//	var err = prim.Scan(value)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	return newObj, nil
	return f.Widget().ValueToGo(value)
}

func (f *ForeignKeyFormField) Widget() widgets.Widget {
	if f.BaseField.FormWidget != nil {
		return f.BaseField.FormWidget
	}

	f.BaseField.FormWidget = ModelSelectWidget(
		f.Field.AllowBlank(),
		"--------",
		chooser.BaseChooserOptions{
			TargetObject: f.Relation.Model(),
			GetPrimaryKey: func(c context.Context, i interface{}) interface{} {
				var def, ok = i.(attrs.Definer)
				if !ok {
					assert.Fail("object %T is not a Definer", i)
				}
				return attrs.PrimaryKey(c, def)
			},
		},
		nil,
	)

	return f.BaseField.FormWidget
}

var _ modelforms.AfterModelFieldSaver = (*ManyToManyFormField)(nil)

type ManyToManyFormField struct {
	BaseRelationField
}

func (o *ManyToManyFormField) SaveField(ctx context.Context, field attrs.Field, value interface{}, model attrs.Definer) error {
	if o.Relation == nil {
		return errors.ValueError.Wrap("relation is nil")
	}

	var defs = attrs.Define(ctx, model)
	var fld, ok = defs.Field(o.Field.Name())
	if !ok {
		return errors.FieldNotFound.Wrapf(
			"Field %q not found in model %T",
			o.Field.Name(), model,
		)
	}

	var backRef = queries.RelM2M[attrs.Definer, attrs.Definer]{
		Parent: &queries.ParentInfo{
			Object: model,
			Field:  fld,
		},
	}

	if value == nil {
		var _, err = backRef.Objects().
			WithContext(ctx).
			ClearTargets()

		if err != nil && !errors.Is(err, errors.NoChanges) {
			return errors.Wrapf(err, "failed to clear targets for %s", o.Field.Name())
		}

		return nil
	}

	objects, ok := value.([]attrs.Definer)
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"Value %v (%T) is not a []attrs.Definer",
			value, value,
		)
	}

	var _, err = backRef.Objects().
		WithContext(ctx).
		SetTargets(objects)
	if err != nil {
		return errors.Wrapf(err, "failed to set targets for %s", o.Field.Name())
	}
	return nil
}

func (o *ManyToManyFormField) Widget() widgets.Widget {
	if o.BaseField.FormWidget != nil {
		return o.BaseField.FormWidget
	}

	o.BaseField.FormWidget = &MultiSelectWidget[attrs.Definer]{
		BaseWidget:   widgets.NewBaseWidget("model-multiple-select", "forms/widgets/model-multiple-select.html", nil),
		IncludeBlank: o.Field.AllowBlank(),
		Relation:     o.Relation,
		FieldDef:     o.Field,
		BlankLabel:   "--------",
	}

	return o.BaseField.FormWidget
}

type OneToOneFormField struct {
	ForeignKeyFormField
}

func (o *OneToOneFormField) SaveField(ctx context.Context, field attrs.Field, value interface{}, model attrs.Definer) error {
	if o.Relation == nil {
		return errors.ValueError.Wrap("relation is nil")
	}

	var defs = attrs.Define(ctx, model)
	var fld, ok = defs.Field(o.Field.Name())
	if !ok {
		return errors.FieldNotFound.Wrapf(
			"Field %q not found in model %T",
			o.Field.Name(), model,
		)
	}

	var backRef = queries.RelO2O[attrs.Definer, attrs.Definer]{
		Parent: &queries.ParentInfo{
			Object: model,
			Field:  fld,
		},
	}

	if value == nil {
		var _, err = backRef.Objects().Clear(ctx)

		if err != nil && !errors.Is(err, errors.NoChanges) {
			return errors.Wrapf(err, "failed to clear targets for %s", o.Field.Name())
		}

		return nil
	}

	objects, ok := value.(attrs.Definer)
	if !ok {
		return errors.TypeMismatch.Wrapf(
			"Value %v (%T) is not a []attrs.Definer",
			value, value,
		)
	}

	var _, _, err = backRef.Objects().
		WithContext(ctx).
		SetTarget(objects)
	if err != nil {
		return errors.Wrapf(err, "failed to set targets for %s", o.Field.Name())
	}
	return nil
}
