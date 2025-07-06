package dbtype

import (
	"encoding/json"
	"fmt"
	"iter"
	"reflect"
	"sync"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-signals"
)

type Type int

const (
	Invalid Type = iota
	Text
	String
	Char
	Int
	Uint
	Float
	Decimal
	Bool
	UUID
	ULID
	Bytes
	JSON
	BLOB
	Timestamp
	LocalTime
	DateTime

	DEFAULT = Text // Default type used when no specific type is registered
)

// CanDBType is an interface that defines a method to get the database type of a value.
//
// It can be implemented by both [attrs.Field], or the [reflect.Type] returned by [attrs.Field.Type]
type CanDBType interface {
	DBType() Type
}

// CanDBTypeString is a type that can be used to define a field that can return a database type as a string.
type CanDBTypeString interface {
	DBType() string
}

var typeNames = map[Type]string{
	Invalid:   "INVALID",
	Text:      "TEXT",
	String:    "STRING",
	Char:      "CHAR",
	Int:       "INT",
	Uint:      "UINT",
	Float:     "FLOAT",
	Decimal:   "DECIMAL",
	Bool:      "BOOL",
	UUID:      "UUID",
	ULID:      "ULID",
	Bytes:     "BYTES",
	JSON:      "JSON",
	BLOB:      "BLOB",
	Timestamp: "TIMESTAMP",
	LocalTime: "LOCALTIME",
	DateTime:  "DATETIME",
}

var typesByName = func() map[string]Type {
	types := make(map[string]Type, len(typeNames))
	for t, name := range typeNames {
		types[name] = t
	}
	return types
}()

func (t Type) String() string {
	if name, exists := typeNames[t]; exists {
		return name
	}
	return "INVALID"
}

func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Type) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}

	if typ, exists := typesByName[name]; exists {
		*t = typ
		return nil
	}

	return fmt.Errorf("unknown db type: %s", name)
}

func NewFromString(s string) (Type, bool) {
	if typ, exists := typesByName[s]; exists {
		return typ, true
	}
	return Invalid, false
}

// The TypeRegistry interface defines methods for managing a registry of database types.
// It is only used for the signal before the type registry is locked.
type TypeRegistry interface {
	Add(srcTyp any, dbType Type, forceKind ...bool) bool
	For(typ reflect.Type) (dbType Type, exists bool)
	IsLocked() bool
	Lock() (success bool)
	Types() iter.Seq2[reflect.Type, Type]
	Unlock()
}

var TYPES = &typeRegistry{
	byType: make(map[reflect.Type]Type),
	byKind: make(map[reflect.Kind]Type),
	Locked: signals.New[TypeRegistry]("dbtype.TYPES.Locked"),
}

// Add a type to the registry.
//
// The srcTyp can be a [reflect.Type], [reflect.Value], [reflect.Kind], or we use reflect.TypeOf(srcTyp) to get the type.
// The dbType is the database type to bind to the srcTyp.
//
// If forceKind is true, the kind of the srcTyp will be used to register the type as well.
//
// After adding the type, calling `For(reflect.Type)` will return the dbType for the srcTyp.
//
// This function will only return false if the type registry is locked.
func Add(srcTyp any, dbType Type, forceKind ...bool) bool {
	return TYPES.Add(srcTyp, dbType, forceKind...)
}

// For returns the database type for the given [reflect.Type].
//
// If the type is not registered, it will return the default type (Text).
func For(typ reflect.Type) (dbType Type, exists bool) {
	return TYPES.For(typ)
}

// Types returns a sequence of all registered types in the registry.
// It yields pairs of [reflect.Type] and [Type], allowing iteration over all registered types.
// This is useful for introspection, debugging purposes and type management.
//
// Currently, it is only used in [drivers.registerTypeConversions] for converting a [string] of `<pkgpath.typename>` to [reflect.Type] for the
// initialization of default values during migrations for older migration files who's default value type might not match the current type.
// this is because the type registry provides a nice and unified way to handle go types which are also used in the database drivers.
func Types() iter.Seq2[reflect.Type, Type] {
	return TYPES.Types()
}

// Lock locks the type registry.
//
// This is used to prevent further modifications to the type registry after it has been locked.
//
// It is used to ensure that the type registry is not modified after everything has been setup and registered.
func Lock() {
	if !TYPES.Lock() {
		// if you are looking at this, and you really need to unlock the type registry,
		// for some reason, please interact with the TYPES variable directly.
		panic("Type registry is already locked, cannot lock again")
	}
}

// IsLocked checks if the type registry is locked.
func IsLocked() bool {
	return TYPES.IsLocked()
}

// typeRegistry is a registry for database types.
//
// It maps reflect.Type and reflect.Kind to Type.
//
// The registry provides a unified way to handle database types
// across different databases and parts of the application.
type typeRegistry struct {
	byType map[reflect.Type]Type
	byKind map[reflect.Kind]Type

	// we use TryLock to avoid deadlocks in some cases - this is
	// so types can still be registered inside of the Locked
	// signal, which is sent when the type registry is locked.
	//
	// we use a mutex to ensure that the locked state is consistent
	// during function calls, using an atomic.Bool would not be enough
	// because the value is read and written multiple times
	mu     sync.Mutex
	locked bool
	Locked signals.Signal[TypeRegistry]
}

func (r *typeRegistry) registerType(typ reflect.Type, dbType Type) {
	if typ == nil {
		return
	}
	if _, exists := r.byType[typ]; exists {
		logger.Warnf(
			"Type %s already registered with type %s, overwriting with %d",
			typ.String(), r.byType[typ], dbType,
		)
	}

	r.byType[typ] = dbType
}

func (r *typeRegistry) registerKind(knd reflect.Kind, dbType Type) {
	if knd == reflect.Invalid {
		return
	}
	if _, exists := r.byKind[knd]; exists {
		logger.Warnf(
			"Kind %s already registered with type %d, overwriting with %d",
			knd.String(), r.byKind[knd], dbType,
		)
	}
	r.byKind[knd] = dbType
}

func (r *typeRegistry) Add(srcTyp any, dbType Type, forceKind ...bool) bool {
	if r.mu.TryLock() {
		defer r.mu.Unlock()
	}

	if r.locked {
		logger.Warn("Type registry is locked, cannot add new types")
		return false
	}

	var (
		typ reflect.Type
		knd reflect.Kind
	)
	switch v := srcTyp.(type) {
	case reflect.Type:
		typ = v
	case reflect.Value:
		typ = v.Type()
	case reflect.Kind:
		knd = v
	default:
		typ = reflect.TypeOf(srcTyp)
	}

	var useKind bool
	if len(forceKind) > 0 {
		useKind = forceKind[0]

		if typ != nil && useKind {
			knd = typ.Kind()
		}
	}

	if typ != nil {
		r.registerType(typ, dbType)
	}

	if useKind || knd != reflect.Invalid {
		r.registerKind(knd, dbType)
	}

	return true
}

func (r *typeRegistry) For(typ reflect.Type) (dbType Type, exists bool) {
	if typ == nil {
		goto retFalse
	}

	if dbType, exists = r.byType[typ]; exists {
		return dbType, true
	}

	if dbType, exists = r.byKind[typ.Kind()]; exists {
		return dbType, true
	}

retFalse:
	return DEFAULT, false
}

func (r *typeRegistry) Types() iter.Seq2[reflect.Type, Type] {
	return func(yield func(reflect.Type, Type) bool) {
		for typ, dbType := range r.byType {
			if !yield(typ, dbType) {
				return
			}
		}
	}
}

func (r *typeRegistry) Lock() bool {
	if r.mu.TryLock() {
		defer r.mu.Unlock()
	}

	if !r.locked {
		assert.Err(r.Locked.Send(r))
		r.locked = true
		return true
	}

	return false
}

func (r *typeRegistry) IsLocked() bool {
	if r.mu.TryLock() {
		defer r.mu.Unlock()
	}
	return r.locked
}

// Unlock releases the lock on the type registry.
//
// This is only used in tests, and thus not exported as a global function.
func (r *typeRegistry) Unlock() {
	if r.mu.TryLock() {
		defer r.mu.Unlock()
	}

	if r.locked {
		r.locked = false
		return
	}

	logger.Warn("Type registry is not locked, cannot unlock")
}
