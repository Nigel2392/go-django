package fields

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type FieldConfig struct {
	DataModelFieldConfig
	ScanTo            any
	Nullable          bool
	AllowEdit         bool
	IsProxy           bool
	IsReverse         any // bool, func() bool, or func(attrs.Field) bool
	ReverseName       string
	NoReverseRelation bool // If true, the field will not create a reverse relation in the model meta.
	ColumnName        string
	TargetField       string
	Through           attrs.Through
	Rel               attrs.Relation
}

type unbound[T attrs.Field] struct {
	name   string
	config *FieldConfig
	field  func(attrs.Definer, string, *FieldConfig) T
}

// Name returns the name of the field.
func (u *unbound[T]) Name() string {
	return u.name
}

// BindField binds the field to the model.
func (u *unbound[T]) BindField(model attrs.Definer) (attrs.Field, error) {
	if u.name == "" {
		panic(fmt.Sprintf("field name cannot be empty for %T", model))
	}

	var fieldConfig = *u.config

	if fieldConfig.ScanTo == nil {
		var (
			rVal = reflect.ValueOf(model)
			rTyp = reflect.TypeOf(model)
		)

		if rVal.Kind() != reflect.Ptr || rTyp.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("model must be a pointer to a struct, got %T", model)
		}

		rTyp = rTyp.Elem()
		rVal = rVal.Elem()

		var field = rVal.FieldByName(u.name)
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found in model %s", u.name, rTyp.Name())
		}

		fieldConfig.ScanTo = field.Addr().Interface()
	}

	var field = u.field(
		model,
		u.name,
		&fieldConfig,
	)

	return field, nil
}

func fieldConstructor[FieldT attrs.Field, T any](name string, fieldFunc func(attrs.Definer, string, *FieldConfig) FieldT, conf ...*FieldConfig) attrs.UnboundFieldConstructor {
	var cnf = &FieldConfig{}
	if len(conf) > 0 {
		cnf = conf[0]
	}

	if cnf.Rel == nil {

		var nT = reflect.TypeOf(new(T)).Elem()
		if nT.Elem().Kind() == reflect.Interface || nT.Kind() == reflect.Ptr {
			nT = nT.Elem()
		}

		var rV = reflect.New(nT)
		var newObject = rV.Interface().(attrs.Definer)
		cnf.Rel = attrs.Relate(
			newObject,
			cnf.TargetField,
			cnf.Through,
		)
	}

	return &unbound[FieldT]{
		name:   name,
		config: cnf,
		field:  fieldFunc,
	}
}

func OneToOne[T any](name string, conf ...*FieldConfig) attrs.UnboundFieldConstructor {
	return fieldConstructor[*OneToOneField[T], T](
		name, NewOneToOneField[T], conf...,
	)
}

func ForeignKey[T any](name, columnName string, conf ...*FieldConfig) attrs.UnboundFieldConstructor {
	if len(conf) == 0 {
		conf = append(conf, &FieldConfig{
			ColumnName: columnName,
		})
	} else if conf[0].ColumnName == "" {
		conf[0].ColumnName = columnName
	}
	return fieldConstructor[*ForeignKeyField[T], T](
		name, NewForeignKeyField[T], conf...,
	)
}

func ManyToMany[T any](name string, conf ...*FieldConfig) attrs.UnboundFieldConstructor {
	if len(conf) == 0 {
		panic("ManyToMany requires at least one FieldConfig with a Through relation defined")
	}

	if conf[0].Rel == nil || (conf[0].Through == nil && conf[0].Rel.Through() == nil) {
		panic("ManyToMany requires a Through relation defined in the FieldConfig")
	}

	return fieldConstructor[*ManyToManyField[T], T](
		name, NewManyToManyField[T], conf...,
	)
}
