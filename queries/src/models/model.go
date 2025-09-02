package models

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"reflect"

	"github.com/Nigel2392/go-django/queries/internal"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-signals"
)

var (
	// Internal interfaces that the model should implement
	_ _ModelInterface  = &Model{}
	_ SaveableObject   = &Model{}
	_ DeleteableObject = &Model{}

	// Third party interfaces that the model should implement
	_ models.ContextSaver                  = &Model{}
	_ queries.CanSetup                     = &Model{}
	_ queries.DataModel                    = &Model{}
	_ queries.Annotator                    = &Model{}
	_ queries.ThroughModelSetter           = &Model{}
	_ queries.ActsAfterSave                = &Model{}
	_ queries.ActsAfterQuery               = &Model{}
	_ attrs.CanSignalChanged               = &Model{}
	_ attrs.CanCreateObject[attrs.Definer] = &Model{}
)

type modelOptions struct {
	// base model information, used to  extract the model / proxy chain
	base   *BaseModelInfo
	object *reflect.Value
	defs   *attrs.ObjectDefinitions
	meta   attrs.ModelMeta
	state  *ModelState
	fromDB bool
}

type proxyModel struct {
	proxy  *BaseModelProxy
	object *Model
}

// The `models` package provides a [Model] struct that is used to represent a model in the GO-Django ORM.
//
// To use the [Model] struct, simply embed it in your own [attrs.Definer] struct.
//
// This will give you access to all the extra functionality provided by the `Model` struct, such as reverse relations, annotations, and the [Model.GetQuerySet] method.
//
// To use the model directly in the code without having fetched it from the database it is recommended to use the [Setup] function on the [attrs.Definer] object,
// passing in the [attrs.Definer] object as an argument.
type Model struct {
	// internals of the model, used to store
	// the model's base information, object, definitions, etc.
	// this is set to nil if the model is not setup yet
	internals *modelOptions

	// changed is a signal which gets emitted when the
	// model is changed (e.g. fields are set, saved, etc.)
	//
	// it is a loose signal, not bound to any specific
	// signal pool. this means it is used only for this model.
	changed signals.Signal[ModelChangeSignal]

	// proxies is a map of proxy models that are bound to the current
	// model. this is used to handle proxy models that are
	// defined in the model's struct.
	proxies map[string]*proxyModel

	// data store for the model, used to store model data
	// like annotations, custom data, etc.
	data queries.ModelDataStore

	// ThroughModel is a model bound to the current
	// object, it will be set if the model is a
	// target of a ManyToMany or OneToMany relation
	// with a through model.
	ThroughModel attrs.Definer

	// annotations for the model, used to store
	// database annotation key value pairs
	Annotations map[string]any
}

// Setup sets up a [attrs.Definer] object so that it's model is properly initialized.
//
// This method is normally called automatically, but when manually defining a struct
// as a model, this method should be called to ensure the model is properly initialized.
//
// In short, this must be called if the model is not created using [attrs.NewObject].
func Setup[T attrs.Definer](def T) T {
	var model, err = ExtractModel(def)
	assert.True(
		err == nil,
		"failed to extract model from definer %T: %v", def, err,
	)
	assert.False(
		model == nil,
		"model is nil, cannot setup model for definer %T", def,
	)

	err = model.Setup(def)
	assert.True(
		err == nil,
		"failed to setup model %T: %v", def, err,
	)
	return def
}

func (m *Model) __Model() private { return private{} }

// checkValid checks if the model is valid and initialized.
func (m *Model) checkValid() {
	assert.False(m.internals == nil,
		fmt.Errorf("model internals are not initialized: %w", ErrModelInitialized),
	)
	assert.False(m.internals.base == nil,
		fmt.Errorf("model base information is not set: %w", ErrModelInitialized),
	)
	assert.False(m.internals.object == nil,
		fmt.Errorf("model object is not set: %w", ErrModelInitialized),
	)
}

func (m *Model) setupInitialState() {
	if m.internals.defs == nil {
		// if the model definitions are not set, we cannot setup the state
		// a nil state assumes that the model is always changed.
		return
	}

	m.internals.state = initState(m)
}

// onChange is a callback that is called when the model changes.
// It is used to handle changes in the model's fields to update the model's state
// when the signal is emitted.
func (m *Model) onChange(s signals.Signal[ModelChangeSignal], ms ModelChangeSignal) error {
	m.checkValid()

	if ms.Model != m {
		panic(fmt.Errorf(
			"model signal %T is not for model %T (%p != %p)",
			ms.Model, m, ms.Model, m,
		))
	}

	if m.internals.state == nil {
		m.setupInitialState()
	}

	// fmt.Printf(
	// "[onChange] Model %T received signal %s for field %s with flags %v (%v)\n",
	// m.internals.object.Interface(),
	// s.Name(), ms.Field.Name(), ms.Flags, ms.Field.GetValue(),
	// )

	switch {
	case ms.Flags.True(FlagModelReset), ms.Flags.True(FlagModelSetup):
		// set the model's initial state
		m.setupInitialState()
		// fmt.Printf(
		// 	"Proxy model %T has been reset or setup, initial state is now set\n",
		// 	m.internals.object.Interface(),
		// )

	case ms.Flags.True(FlagProxyChanged):
		m.internals.state.change(ms.StructField.Name)

		// fmt.Printf(
		// 	"Model %T proxy field %s changed to %v\n",
		// 	m.internals.object.Interface(),
		// 	fieldName, ms.Model.internals.object.Interface(),
		// )

	case ms.Flags.True(FlagFieldChanged):
		m.internals.state.change(ms.Field.Name())

	// fmt.Printf(
	// 	"Model %T field %s changed to %v\n",
	// 	m.internals.object.Interface(),
	// 	ms.Field.Name(), ms.Field.GetValue(),
	// )

	case ms.Flags.True(FlagFieldReset):
		m.internals.state.reset(ms.Field.Name())

	default:
		// if the signal is not for a field change, we can skip it
		panic(fmt.Errorf(
			"model signal %T is not for a field change, flags: %v",
			ms.Model, ms.Flags,
		))
	}

	return nil
}

// SignalChanged sends a signal that the model has changed.
//
// This is used to allow the [attrs.Definitions] to callback to the model
// and notify it that the model has changed, so it can update its internal state
// and trigger any necessary updates.
func (m *Model) SignalChange(fa attrs.Field, value interface{}) {
	m.checkValid()

	//	fmt.Printf(
	//		"[SignalChange] Model %T field %s changed to %v\n",
	//		m.internals.object.Interface(),
	//		fa.Name(), value,
	//	)

	m.changed.Send(ModelChangeSignal{
		Model:  m,
		Field:  fa,
		Flags:  FlagFieldChanged,
		Object: m.internals.object.Interface().(attrs.Definer),
	})
}

// SignalReset is called when a field's changed status should be reset.
func (m *Model) SignalReset(fa attrs.Field) {
	m.checkValid()

	m.changed.Send(ModelChangeSignal{
		Model:  m,
		Field:  fa,
		Flags:  FlagFieldReset,
		Object: m.internals.object.Interface().(attrs.Definer),
	})
}

// State returns the current state of the model.
//
// The state is initialized when the model is setup,
// and it contains the initial values of the model's fields
// as well as the changed fields.
func (m *Model) State() *ModelState {
	m.checkValid()
	if m.internals.state == nil {
		m.setupInitialState()
	}
	return m.internals.state
}

// CreateObject creates a new object of the model type
// and sets it up with the model's definitions.
//
// It returns nil if the object is not valid or if the model
// is not registered with the model system.
//
// This automatically sets up the model's fields
// and handles the proxy model if it exists.
//
// This method is automatically called by the
// [attrs.NewObject] function when a new object is created.
func (m *Model) CreateObject(object attrs.Definer) attrs.Definer {
	if !attrs.IsModelRegistered(object) {
		return nil
	}

	var obj = reflect.ValueOf(object)
	if !obj.IsValid() {
		return nil
	}

	if obj.IsNil() {
		obj = reflect.New(obj.Type().Elem())
		object = obj.Interface().(attrs.Definer)
	}

	var base = getModelChain(object)
	if base == nil {
		return nil
	}

	var newObj = reflect.New(obj.Type().Elem())
	var modelVal = newObj.Elem().FieldByIndex(
		base.base.Index,
	)

	var model = modelVal.Addr().Interface().(*Model)
	if err := model.Setup(newObj.Interface().(attrs.Definer)); err != nil {
		return nil
	}

	var newDefiner = newObj.Interface().(attrs.Definer)
	if len(base.base.Index) > 1 {
		for i := 0; i < len(base.base.Index)-1; i++ {
			if i == len(base.base.Index)-1 {
				// if we are at the last index - it is the [Model] itself.
				// we do not implement the [attrs.Embedded] interface.
				break
			}

			var field = newObj.Elem().FieldByIndex(base.base.Index[:i+1])
			if field.Addr().Type().Implements(reflect.TypeOf((*attrs.Embedded)(nil)).Elem()) {
				var setupObj = field.Addr().Interface().(attrs.Embedded)
				if err := setupObj.BindToEmbedder(newDefiner); err != nil {
					assert.Fail(
						"failed to bind embedded object %T to embedder %T: %v",
						setupObj, newDefiner, err,
					)
					return newDefiner
				}
			}
		}
	}

	return newDefiner
}

func (m *Model) Setup(def attrs.Definer) error {
	if def == nil {
		return ErrObjectInvalid
	}

	// Retrieve the pre-compiled model chain
	var base = getModelChain(def)
	if base == nil {
		return fmt.Errorf(
			"object %T does not have an embedded Model field: %w",
			def, ErrModelEmbedded,
		)
	}

	// check if the object's model field is
	// points to the current model
	var defValue = reflect.ValueOf(def)
	var defElem = defValue.Elem()
	var self = defElem.FieldByIndex(base.base.Index)
	if self.Addr().Pointer() != reflect.ValueOf(m).Pointer() {
		return fmt.Errorf(
			"object %T is not the same as the model %T, expected %d, got %d",
			def, m, self.Addr().Pointer(), reflect.ValueOf(m).Pointer(),
		)
	}

	var sig = ModelSignal{
		SignalInfo: ModelSignalInfo{
			Data: make(map[string]any),
		},
		Model:  m,
		Object: def,
	}

	// Handle the model's proxy object if it exists.
	var changedProxies, err = m.setupProxy(
		base,
		defValue,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to setup proxy for model %T: %w",
			def, err,
		)
	}

	// if the proxy was changed it needs
	// to be reset, we need to clear the internals
	// as some fields may be pointing to the old object
	if len(changedProxies) > 0 && m.internals != nil {
		sig.SignalInfo.Flags.set(FlagProxySetup)
		m.internals.object = nil
		m.internals.defs = nil
		m.changed = nil
	}

	// validate if it is the same object
	// if not, clear the defs so any old fields pointing to the old
	// object will be cleared
	if (m.internals != nil && m.internals.defs != nil) && (m.internals.object != nil && m.internals.object.Pointer() != defValue.Pointer()) {
		sig.SignalInfo.Flags.set(FlagModelReset)
		sig.SignalInfo.Data["old"] = m.internals.defs.Object
		sig.SignalInfo.Data["new"] = def
		m.internals.defs = nil
		m.internals.object = nil
		m.changed = nil
	}

	// no changes were made, pointers equal according to above check
	if len(changedProxies) == 0 && m.internals != nil && m.internals.object != nil {
		return nil
	}

	if m.changed == nil {
		m.changed = signals.New[ModelChangeSignal]("model.changed")
		m.changed.Listen(m.onChange)
	}

	// if the model is not setup, we need to initialize it
	if m.internals == nil || m.internals.object == nil {
		sig.SignalInfo.Flags.set(FlagModelSetup)
		m.internals = &modelOptions{
			object: &defValue,
			base:   base,
		}
	}

	// send the model setup signal
	if !sig.SignalInfo.Flags.True(ModelSignalFlagNone) {
		if err := SIGNAL_MODEL_SETUP.Send(sig); err != nil {
			return fmt.Errorf(
				"failed to emit model setup signal for %T: %w",
				def, err,
			)
		}
	}

	return nil
}

// setupProxy sets up the proxy for the model if it exists.
// It checks if the proxy field is set, and if so, it extracts the
// embedded model from the proxy field and calls Setup on it with the
// provided definer proxy object.
func (m *Model) setupProxy(base *BaseModelInfo, parent reflect.Value) (changedList []string, err error) {
	if len(base.proxies) == 0 {
		return nil, nil
	}

	if parent.Kind() == reflect.Ptr {
		parent = parent.Elem()
	}

	if m.proxies == nil {
		m.proxies = make(map[string]*proxyModel)
	}

	changedList = make([]string, 0, len(base.proxies))
	for _, proxy := range base.proxies {
		var (
			rVal          = parent.FieldByIndex(proxy.rootField.Index)
			newValueIsNil = rVal.IsNil()
		)

		var currentProxy = m.proxies[proxy.rootField.Name]
		if currentProxy == nil {
			currentProxy = &proxyModel{
				proxy:  proxy,
				object: nil,
			}
			m.proxies[proxy.rootField.Name] = currentProxy
		}

		var (
			currentProxyIsNil = currentProxy.object == nil
			nilDiff           = !newValueIsNil && currentProxyIsNil
			ptrDiff           = (!currentProxyIsNil && currentProxy.object.internals.object.Pointer() != rVal.Pointer())
			changed           = nilDiff || ptrDiff
		)

		// if the proxy is nil, we need to create a new one when specified
		if newValueIsNil && (proxy.directField.Tag.Get("auto") == "true" || proxy.rootField.Tag.Get("auto") == "true") {
			var newObj = attrs.NewObject[attrs.Definer](proxy.rootField.Type)
			rVal.Set(reflect.ValueOf(newObj))
			newValueIsNil = false
			changed = true
		}

		if !changed {
			// if the proxy is not changed, we can skip the setup
			// and return early
			continue
		}

		if rVal.IsNil() && changed && !currentProxyIsNil {
			changed = true
			currentProxy.object = nil
			m.internals.defs = nil
			changedList = append(changedList, proxy.rootField.Name)
			continue
		}

		// if there is a difference in the pointer or one of
		// the pointers is nil, we need to reset the proxy
		if !rVal.IsNil() && changed {
			changed = true
			var proxyObj = rVal.Interface().(attrs.Definer)
			var modelValue = rVal.Elem().FieldByIndex(
				proxy.next.base.Index,
			)
			var modelPtr = modelValue.Addr().Interface()
			currentProxy.object = modelPtr.(*Model)

			// proxy object must not be nil
			if currentProxy.object == nil {
				return nil, fmt.Errorf(
					"failed to extract embedded model from proxy field %s: %w",
					proxy.rootField.Name, ErrObjectInvalid,
				)
			}

			// setup the proxy object
			err = currentProxy.object.Setup(proxyObj)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to setup embedded model from proxy field %s: %w",
					proxy.rootField.Name, err,
				)
			}

			currentProxy.object.changed.Listen(func(s signals.Signal[ModelChangeSignal], ms ModelChangeSignal) error {
				m.changed.Send(ModelChangeSignal{
					Flags:       FlagProxyChanged,
					StructField: proxy.rootField,
					Next:        &ms,
					Model:       m,
				})
				return nil
			})
		}

		if changed {
			changedList = append(changedList, proxy.rootField.Name)
		}
	}

	return changedList, nil
}

// Define defines the fields of the model based on the provided definer
//
// Normally this would be done with [attrs.Define], the current model method
// is a convenience method which also handles the setup of the model
// as well as reverse relation setup.
func (m *Model) Define(def attrs.Definer, flds ...any) *attrs.ObjectDefinitions {
	if err := m.Setup(def); err != nil {
		panic("failed to setup model: " + err.Error())
	}

	m.checkValid()

	if m.internals.defs == nil {

		var tableName = m.internals.base.base.Tag.Get("table")
		var _fieldsIter = attrs.UnpackFieldsFromArgsIter(def, flds...)
		var _fields = make([]attrs.Field, 0, 2)
		var meta = attrs.GetModelMeta(def)
		for head := meta.ReverseMap().Front(); head != nil; head = head.Next() {
			var (
				field attrs.Field

				key   = head.Key
				value = head.Value
				typ   = value.Type()
				from  = value.From()
			)

			//	if reflect.TypeOf(value.Model()) == reflect.TypeOf(def) {
			//		panic(fmt.Errorf(
			//			"reverse relation %q in model %T is not allowed to point to itself",
			//			key, def,
			//		))
			//	}

			var fromModelField = from.Field()
			if fromModelField == nil {
				panic(fmt.Errorf(
					"reverse relation %q in model %T does not have a field defined",
					key, def,
				))
			}

			var conf = &fields.FieldConfig{
				ScanTo:      def,
				ReverseName: key,
				ColumnName:  fromModelField.ColumnName(),
				Rel:         value,
			}

			switch typ {
			case attrs.RelOneToOne: // OneToOne
				if head.Value.Through() == nil {
					field = fields.NewOneToOneReverseField[attrs.Definer](def, key, conf)
				} else {
					field = fields.NewOneToOneReverseField[queries.Relation](def, key, conf)
				}
			case attrs.RelManyToOne: // ManyToOne, ForeignKey
				field = fields.NewForeignKeyField[attrs.Definer](def, key, conf)
			case attrs.RelOneToMany: // OneToMany, ForeignKeyReverse
				field = fields.NewForeignKeyReverseField[*queries.RelRevFK[attrs.Definer]](def, key, conf)
			case attrs.RelManyToMany: // ManyToMany
				field = fields.NewManyToManyField[*queries.RelM2M[attrs.Definer, attrs.Definer]](def, key, conf)
			default:
				panic("unknown relation type: " + typ.String())
			}

			if field != nil {
				_fields = append(_fields, field)
			}
		}

		for _, proxy := range m.internals.base.proxies {
			var (
				// create a new plain proxy object to use as target in the relation
				field        attrs.Field
				fieldName    = proxy.rootField.Name
				rNewProxyObj = reflect.New(proxy.directField.Type.Elem())
				newProxyObj  = rNewProxyObj.Interface().(attrs.Definer)
			)

			switch {
			case proxy.cTypeFieldName == "":
				// assume target field is set, ctype is not set
				// use o2o with target field
				field = fields.NewOneToOneField[attrs.Definer](def, fieldName, &fields.FieldConfig{
					ScanTo:      def,
					IsProxy:     true,
					AllowEdit:   true,
					TargetField: proxy.targetFieldName,
					Rel:         attrs.Relate(newProxyObj, proxy.targetFieldName, nil),
					DataModelFieldConfig: fields.DataModelFieldConfig{
						ResultType: rNewProxyObj.Type(),
					},
				})
			case proxy.cTypeFieldName != "" && proxy.targetFieldName != "" && !proxy.controlsSaving:
				field = newProxyField(m, def, proxy.rootField.Name, fieldName, &ProxyFieldConfig{
					Proxy:            newProxyObj,
					ContentTypeField: proxy.cTypeFieldName,
					TargetField:      proxy.targetFieldName,
					AllowEdit:        true,
				})

			case proxy.cTypeFieldName != "" && proxy.targetFieldName != "" && proxy.controlsSaving:
				field = newProxyField(m, def, proxy.rootField.Name, fieldName, &ProxyFieldConfig{
					Proxy:            newProxyObj,
					ContentTypeField: proxy.cTypeFieldName,
					TargetField:      proxy.targetFieldName,
				})
			default:
				panic(fmt.Errorf(
					"proxy %s in model %T does not have a content type field or target field defined",
					proxy.rootField.Name, def,
				))
			}

			// add the proxy field to the model definitions
			_fields = append(_fields, field)
		}

		m.internals.defs = attrs.Define[attrs.Definer, any](
			def, _fieldsIter, _fields,
		)

		if tableName != "" && m.internals.defs.Table == "" {
			m.internals.defs.Table = tableName
		}
	}

	return m.internals.defs
}

// Defs returns the model's definitions.
//
// If the model is not properly initialized it will panic.
func (m *Model) Defs() *attrs.ObjectDefinitions {
	m.checkValid()
	return m.internals.defs
}

// PK returns the primary key field of the model.
//
// If the model is not properly initialized it will panic.
//
// If the model does not have a primary key defined, it will return nil.
func (m *Model) PK() attrs.Field {
	m.checkValid()

	if m.internals.defs == nil {
		return nil
	}

	return m.internals.defs.Primary()
}

// Object returns the object of the model.
//
// It checks if the model is properly initialized and if the object is set up,
// if the model is not properly set up, it will panic.
func (m *Model) Object() attrs.Definer {
	m.checkValid()
	return m.internals.object.Interface().(attrs.Definer)
}

// ModelMeta returns the model's metadata.
func (m *Model) ModelMeta() attrs.ModelMeta {
	m.checkValid()
	if m.internals.meta == nil {
		m.internals.meta = attrs.GetModelMeta(*m.internals.object)
	}
	return m.internals.meta
}

// Saved checks if the model is saved to the database.
// It checks if the model is properly initialized and if the model's definitions
// are set up. If the model is not initialized, it returns false.
// If the model is initialized, it checks if the model was loaded from the database
// or if the primary key field is set. If the primary key field is nil, it returns false.
// If the primary key field has a value, it returns true.
func (m *Model) Saved() bool {
	// if the model is not initialized, it is assumed
	// that it is not saved, so we return false
	if m.internals == nil ||
		m.internals.base == nil ||
		m.internals.object == nil {
		return false
	}

	// if the model was loaded from the database, it is saved
	if m.internals.fromDB {
		return true
	}

	// if the model has a nil primary key field,
	// we assume it is not saved.
	var pk = m.PK()
	if pk == nil {
		return false
	}

	var value, err = pk.Value()
	if err != nil {
		// if we cannot get the value of the primary key,
		// we assume it is not saved
		return false
	}

	return !attrs.IsZero(value)
}

// AfterQuery is called after a query is executed on the model.
//
// This is useful for setup after the model has been loaded from the database,
// such as setting the initial state of the model and marking it as loaded from the database.
func (m *Model) AfterQuery(ctx context.Context) error {
	m.checkValid()
	m.setupInitialState()
	m.internals.fromDB = true
	return nil
}

// AfterSave is called after the model is saved to the database.
//
// This is useful for setup after the model has been saved to the database,
// such as setting the initial state of the model and marking it as loaded from the database.
func (m *Model) AfterSave(ctx context.Context) error {
	m.checkValid()
	m.internals.fromDB = true
	return nil
}

// If this model was the target end of a through relation,
// this method will set the through model for this model.
func (m *Model) SetThroughModel(throughModel attrs.Definer) {
	m.ThroughModel = throughModel
}

// Annotate adds annotations to the model.
// Annotations are key-value pairs that can be used to store additional
// information about the model, such as database annotations or custom data.
func (m *Model) Annotate(annotations map[string]any) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]any)
	}

	maps.Copy(m.Annotations, annotations)
}

// DataStore returns the data store for the model.
//
// The data store is used to store model data like annotations, custom data, etc.
// If the data store is not initialized, it will be created.
func (m *Model) DataStore() queries.ModelDataStore {
	if m.data == nil {
		m.data = make(MapDataStore)
	}
	return m.data
}

// Validate checks if the model is valid.
func (m *Model) Validate(ctx context.Context) error {
	return nil
}

// Save saves the model to the database.
//
// It checks if the model is properly initialized and if the model's definitions
// are set up. If the model is not initialized, it returns an error.
//
// If the model is initialized, it calls the SaveObject method on the model's
// object, passing the current context and a SaveConfig struct that contains
// the model's object, query set, fields to save, and a force flag.
//
// The object embedding the model can choose to implement the
// [canSaveObject] interface to provide a custom save implementation.
func (m *Model) Save(ctx context.Context) error {
	if m.internals == nil || m.internals.object == nil {
		return errors.NotImplemented.WithCause(fmt.Errorf(
			"cannot save model %w", ErrModelInitialized,
		))
	}

	var config SaveConfig
	if cnf, ok := saveConfigFromContext(ctx); ok {
		config = cnf
	}

	var this = m.internals.object.Interface().(attrs.Definer)
	config.this = this

	// This should be true, mostly - UNLESS a model
	// embeds another model that implements the
	// SaveableObject interface, in which case
	// this will turn into an ambiguous method call,
	// thus it cannot be done - this also means the object likely
	// does not override the SaveObject method, meaning it is safe
	// to call the default [Model.SaveObject] method.
	if saveAble, ok := this.(SaveableObject); ok {
		return saveAble.SaveObject(ctx, config)
	}

	logger.Debugf(
		"Model %T does not implement the `SaveableObject` interface, using default save method",
		this,
	)
	return m.SaveObject(ctx, config)
}

// Create saves the model to the database as a new object.
//
// It checks if the model is properly initialized and if the model's definitions
// are set up. If the model is not initialized, it returns an error.
func (m *Model) Create(ctx context.Context) error {
	if m.internals == nil || m.internals.object == nil {
		return errors.NotImplemented.WithCause(fmt.Errorf(
			"cannot save model %w", ErrModelInitialized,
		))
	}

	var config SaveConfig
	if cnf, ok := saveConfigFromContext(ctx); ok {
		config = cnf
	}

	var this = m.internals.object.Interface().(attrs.Definer)
	config.this = this
	config.ForceCreate = true

	// See the comment in [Model.Save] for more information
	// about a possibly ambiguous method call.
	if saveAble, ok := this.(SaveableObject); ok {
		return saveAble.SaveObject(ctx, config)
	}

	logger.Debugf(
		"Model %T does not implement the `SaveableObject` interface, using default save method",
		this,
	)
	return m.SaveObject(ctx, config)
}

// Update saves the model to the database as an update operation.
//
// It checks if the model is properly initialized and if the model's definitions
// are set up. If the model is not initialized, it returns an error.
func (m *Model) Update(ctx context.Context) error {
	if m.internals == nil || m.internals.object == nil {
		return errors.NotImplemented.WithCause(fmt.Errorf(
			"cannot save model %w", ErrModelInitialized,
		))
	}

	var config SaveConfig
	if cnf, ok := saveConfigFromContext(ctx); ok {
		config = cnf
	}

	var this = m.internals.object.Interface().(attrs.Definer)
	config.this = this
	config.ForceUpdate = true

	// See the comment in [Model.Save] for more information
	// about a possibly ambiguous method call.
	if saveAble, ok := this.(SaveableObject); ok {
		return saveAble.SaveObject(ctx, config)
	}

	logger.Debugf(
		"Model %T does not implement the `SaveableObject` interface, using default save method",
		this,
	)
	return m.SaveObject(ctx, config)
}

// Delete deletes the model from the database.
func (m *Model) Delete(ctx context.Context) error {
	if m.internals == nil || m.internals.object == nil {
		return errors.NotImplemented.WithCause(fmt.Errorf(
			"cannot delete model %w", ErrModelInitialized,
		))
	}

	var this = m.internals.object.Interface().(attrs.Definer)

	// See the comment in [Model.Save] for more information
	// about a possibly ambiguous method call.
	if deleteAble, ok := this.(DeleteableObject); ok {
		return deleteAble.DeleteObject(ctx)
	}

	logger.Debugf(
		"Model %T does not implement the `DeleteableObject` interface, using default delete method",
		this,
	)
	return m.DeleteObject(ctx)
}

type saveConfigContextKey struct{}

func NewSaveConfigContext(SaveConfig SaveConfig) context.Context {
	return SaveConfigContext(context.Background(), SaveConfig)
}

func SaveConfigContext(ctx context.Context, cnf SaveConfig) context.Context {
	return context.WithValue(ctx, saveConfigContextKey{}, cnf)
}

func saveConfigFromContext(ctx context.Context) (SaveConfig, bool) {
	var cnf, ok = ctx.Value(saveConfigContextKey{}).(SaveConfig)
	if !ok {
		return SaveConfig{}, false
	}

	return cnf, true
}

type SaveConfig struct {
	// this should not be nil, it is the object itself.
	//
	// If not provided, it will be set to the model's object inside of [Model.SaveObject].
	this attrs.Definer

	// A custom queryset to use for creating or updating the model.
	QuerySet *queries.QuerySet[attrs.Definer]

	// Fields to save, if empty, all fields will be saved.
	// If the model is not loaded from the database, all fields will be saved.
	Fields []string

	// IncludeFields are fields which must be saved, even if they have not changed
	// according to the model's state.
	// This is used to force the save operation to include fields
	// that are not changed, but must be saved for some reason.
	IncludeFields []string

	ForceCreate bool
	ForceUpdate bool
}

func (cnf SaveConfig) Force() bool {
	// if ForceCreate or ForceUpdate is set, we force the save operation
	return cnf.ForceCreate || cnf.ForceUpdate
}

func (cnf SaveConfig) fields() []string {
	// if the fields are not set, we return all fields
	if len(cnf.Fields) == 0 && len(cnf.IncludeFields) == 0 {
		return nil
	}

	// if the fields are set, we return them
	var fields = make([]string, 0, len(cnf.Fields)+len(cnf.IncludeFields))
	fields = append(fields, cnf.Fields...)
	fields = append(fields, cnf.IncludeFields...)
	return fields
}

// SaveObject saves the model's object to the database.
//
// It checks if the model is properly initialized and if the model's definitions
// are set up. If the model is not initialized, it returns an error.
//
// If the model is initialized, it iterates over the model's fields and checks
// if any of the fields have changed. If any field has changed, it adds the field
// to the list of changed fields and prepares a queryset to save the model.
//
// A config struct [SaveConfig] is used to pass the model's object, queryset, fields to save,
// and a force flag to indicate whether to force the save operation.
func (m *Model) SaveObject(ctx context.Context, cnf SaveConfig) (err error) {
	if m.internals == nil || m.internals.object == nil {
		return fmt.Errorf(
			"cannot save fields for %T: %w",
			m.internals.object.Interface(),
			ErrModelInitialized,
		)
	}

	if m.internals.defs == nil {
		var obj = m.internals.object.Interface().(attrs.Definer)
		obj.FieldDefs()
	}

	// Setup the "this" object if not provided.
	if cnf.this == nil {
		cnf.this = m.internals.object.Interface().(attrs.Definer)
	}

	// Create an actor for the model,
	// if the model does not implement the
	// actor interfaces, this is a no-op.
	var actor = queries.Actor(cnf.this)
	ctx, err = actor.BeforeSave(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to run BeforeSave for model %T: %w",
			cnf.this, err,
		)
	}

	// check if anything has changed,
	var fields = internal.NewSet(cnf.fields()...)
	if m.internals.state == nil && m.internals.fromDB && !cnf.Force() && len(fields) == 0 {
		// if the model was loaded from the database and no fields have changed and the state is nil,
		// we can skip saving
		return nil
	}

	if !m.internals.state.Changed(true) && m.internals.fromDB && !cnf.Force() && len(fields) == 0 {
		// if nothing has changed, we can skip saving
		return nil
	}

	if m.internals.defs == nil || m.internals.defs.ObjectFields == nil {
		assert.Fail("Model %T is not properly initialized", m.internals.object.Interface())
	}

	// Start transaction, if one was already started this is a no-op.
	var transaction drivers.Transaction
	if queries.QUERYSET_CREATE_IMPLICIT_TRANSACTION {
		ctx, transaction, err = queries.StartTransaction(ctx)
		if err != nil {
			return fmt.Errorf(
				"failed to start transaction for model %T: %w",
				m.internals.object.Interface(), err,
			)
		}
	} else {
		transaction = queries.NullTransaction()
	}
	defer transaction.Rollback(ctx)

	var (
		// if the model was not loaded from the database,
		// we automatically assume all changes are to be saved
		anyChanges = !m.internals.fromDB

		// Setup to save fields / select relevant fields to update.
		selectFields   = make([]interface{}, 0)
		saveBeforeSelf = make([]queries.SaveableField, 0, m.internals.defs.ObjectFields.Len())
		saveAfterSelf  = make([]queries.SaveableDependantField, 0, m.internals.defs.ObjectFields.Len())
	)
	for head := m.internals.defs.ObjectFields.Front(); head != nil; head = head.Next() {

		// if there was a list of fields provided and if
		// the field is not in the list of fields to save, we skip it
		var mustInclField bool
		if len(fields) > 0 {
			if !fields.Contains(head.Value.Name()) && !cnf.Force() && m.internals.fromDB {
				continue
			}
			mustInclField = true
		}

		// No changes were made to the field, we can skip it.
		var hasChanged = m.internals.state.HasChanged(head.Value.Name())
		if !hasChanged && !mustInclField && !cnf.Force() && m.internals.fromDB {
			continue
		}

		if attrs.IsEmbeddedField(head.Value) {
			// if the field is an embedded field, we skip it
			// as it is not a field that can be saved directly
			// and it will be handled by the parent model.
			continue
		}

		if err := head.Value.Validate(); err != nil {
			return fmt.Errorf(
				"failed to validate field %s in model %T: %w",
				head.Value.Name(), cnf.this, err,
			)
		}

		//	fmt.Printf(
		//		"[SaveObject] Model %T field %s changed: %v, must include: %v, force: %v, value: %#v\n",
		//		cnf.this, head.Value.Name(), hasChanged, mustInclField, cnf.Force, head.Value.GetValue(),
		//	)

		// Check if the field is a Saver or a SaveableField.
		// If it is a Saver, we need to panic and inform the user
		// that they need to use a ContextSaver to maintain transaction integrity.
		switch fld := head.Value.(type) {
		case models.Saver:
			panic(fmt.Errorf(
				"model %T field %s is a Saver, which is not supported in Save(), a ContextSaver is required to maintain transaction integrity",
				cnf.this, head.Value.Name(),
			))
		case queries.SaveableField:
			saveBeforeSelf = append(saveBeforeSelf, fld)
		case queries.SaveableDependantField:
			saveAfterSelf = append(saveAfterSelf, fld)
		}

		// Add the field name to the list of changed fields.
		// This is used to determine which fields to save in the query set.
		selectFields = append(selectFields, head.Value.Name())
		anyChanges = true
	}

	validator, ok := m.internals.object.Interface().(queries.ContextValidator)
	if ok {
		if err := validator.Validate(ctx); err != nil {
			return fmt.Errorf(
				"failed to validate model %T: %w",
				cnf.this, err,
			)
		}
	}

	// if no changes were made and the force flag is not set,
	// we can skip saving the model
	if !anyChanges && !cnf.Force() && len(cnf.Fields) == 0 && m.internals.fromDB {
		return nil
	}

	// Save fields which do not depend on the model itself,
	// these are fields that can be / should be saved before the model itself is saved.
	for _, field := range saveBeforeSelf {
		if err := saveField(ctx, &cnf, field, saveRegularField); err != nil {
			return errors.SaveFailed.WithCause(fmt.Errorf(
				"failed to save field %q in model %T: %w",
				field.Name(), cnf.this, err,
			))
		}
	}

	// Setup the query set if not provided.
	var querySet = cnf.QuerySet
	if querySet == nil {
		querySet = queries.
			GetQuerySet(cnf.this).
			Select(selectFields...).
			ExplicitSave()
	}

	// Add the context to the query set.
	querySet = querySet.
		WithContext(ctx)

	// Perform the save operation on the model.
	// If the model is already saved, it will update the model,
	var updated int64
	var saved = m.Saved()
	if saved && !cnf.ForceCreate || cnf.ForceUpdate {
		updated, err = querySet.Update(cnf.this)
	} else {
		_, err = querySet.Create(cnf.this)
	}
	if err != nil {
		var s = "create"
		if saved && !cnf.ForceCreate || cnf.ForceUpdate {
			s = "update"
		}
		return errors.SaveFailed.WithCause(fmt.Errorf(
			"failed to %s model %T: %w",
			s, m.internals.object.Interface(), err,
		))
	}

	// If no changes were made and the model was saved,
	// we return an error indicating that no rows were affected.
	if (saved || cnf.ForceUpdate) && updated == 0 && !cnf.ForceCreate {
		// sql.ErrNoRows
		return errors.NoChanges.WithCause(fmt.Errorf(
			"model %T was not saved: %w",
			m.internals.object.Interface(), sql.ErrNoRows,
		))
	}

	// Save all fields that depend on the model itself,
	// these are fields that should be saved after the model itself is saved.
	// This is useful for fields that depend on the model's primary key or other fields.
	for _, field := range saveAfterSelf {
		if err := saveField(ctx, &cnf, field, saveDependantField); err != nil {
			return err
		}
	}

	// reset the state after saving
	m.internals.state.Reset()
	m.internals.fromDB = true

	return transaction.Commit(ctx)
}

func (m *Model) DeleteObject(ctx context.Context) error {
	var this = m.internals.object.Interface().(attrs.Definer)
	var defs = this.FieldDefs()
	var prim = defs.Primary()
	if prim == nil {
		return fmt.Errorf(
			"cannot delete model %T: no primary key defined",
			this,
		)
	}

	where, err := queries.GenerateObjectsWhereClause(this)
	if err != nil {
		return fmt.Errorf(
			"failed to generate where clause for model %T: %w",
			this, err,
		)
	}

	actor := queries.Actor(this)
	ctx, err = actor.BeforeDelete(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to run BeforeDelete for model %T: %w",
			this, err,
		)
	}

	_, err = queries.GetQuerySet(this).
		Filter(where).
		Delete()
	if err != nil {
		return fmt.Errorf(
			"failed to delete model %T: %w",
			this, err,
		)
	}

	// After the delete operation, we can reset the model's state
	m.internals.fromDB = false
	if m.internals.state != nil {
		m.internals.state.Reset()
	}

	_, err = actor.AfterDelete(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to run AfterDelete for model %T: %w",
			this, err,
		)
	}

	return nil
}

func saveRegularField(ctx context.Context, cnf *SaveConfig, field queries.SaveableField) error {
	return field.Save(ctx)
}

func saveDependantField(ctx context.Context, cnf *SaveConfig, field queries.SaveableDependantField) error {
	return field.Save(ctx, cnf.this)
}

func saveField[T attrs.FieldDefinition](ctx context.Context, cnf *SaveConfig, field T, save func(ctx context.Context, cnf *SaveConfig, field T) error) error {
	var err error

	if saver, ok := any(field).(T); ok {
		err = save(ctx, cnf, saver)
	}

	if err != nil {
		if !errors.Is(err, errors.NotImplemented) {
			return fmt.Errorf(
				"failed to save field %q in model %T: %w",
				field.Name(), cnf.this, err,
			)
		}

		logger.Warnf(
			"field %q in model %T is not saveable, skipping: %v",
			field.Name(), cnf.this, err,
		)
	}

	return nil
}
