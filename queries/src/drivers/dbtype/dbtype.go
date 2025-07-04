package dbtype

import (
	"encoding/json"
	"fmt"
	"iter"
	"reflect"
	"sync/atomic"

	"github.com/Nigel2392/go-django/src/core/logger"
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
	return "UNKNOWN"
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

var TYPES = typeRegistry{
	byType: make(map[reflect.Type]Type),
	byKind: make(map[reflect.Kind]Type),
}

func Add(srcTyp any, dbType Type, forceKind ...bool) bool {
	return TYPES.Add(srcTyp, dbType, forceKind...)
}

func For(typ reflect.Type) (dbType Type, exists bool) {
	return TYPES.For(typ)
}

func Types() iter.Seq2[reflect.Type, Type] {
	return TYPES.Types()
}

func Lock() {
	if TYPES.locked.CompareAndSwap(false, true) {
		return
	}
	// logger.Warn("Type registry is already locked, cannot modify types")
	panic("Type registry is already locked, cannot modify types")
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
	locked atomic.Bool
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

	if r.locked.Load() {
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

func (r *typeRegistry) Lock() {
	if r.locked.CompareAndSwap(false, true) {
		return
	}
	logger.Warn("Type registry is already locked, cannot modify types")
}

func (r *typeRegistry) IsLocked() bool {
	return r.locked.Load()
}

// Unlock releases the lock on the type registry.
//
// This is only used in tests, and thus not exported as a global function.
func (r *typeRegistry) Unlock() {
	if r.locked.CompareAndSwap(true, false) {
		return
	}
	logger.Warn("Type registry is not locked, cannot unlock")
}
