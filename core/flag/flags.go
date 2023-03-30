package flag

import (
	"flag"
	"os"
	"reflect"
)

// Allowed is the allowed types for a command.
//
// This is for type assertion of the flag, but remains unused.
type Allowed interface {
	~string | ~bool | ~int64 | ~int | ~uint64 | ~uint | ~float64
}

// A wrapper for the std.FlagSet.
type Flags struct {
	Commands []*Command
	errors   ErrorMap
	FlagSet  *flag.FlagSet
	Info     string
	wasRan   bool
}

// ErrorHandling is the error handling for the flag set.
type ErrorHandling flag.ErrorHandling

const (
	// See std.FlagSet. ContinueOnError is the default.
	ContinueOnError = ErrorHandling(flag.ContinueOnError)
	ExitOnError     = ErrorHandling(flag.ExitOnError)
	PanicOnError    = ErrorHandling(flag.PanicOnError)
)

// Initialize a new flag set.
func NewFlags(name string, handling ...ErrorHandling) *Flags {
	var errHandler ErrorHandling = ContinueOnError
	if len(handling) > 0 {
		errHandler = handling[0]
	}

	var f = &Flags{
		Commands: make([]*Command, 0),
		errors:   make(ErrorMap, 0),
		FlagSet:  flag.NewFlagSet(name, flag.ErrorHandling(errHandler)),
	}

	f.FlagSet.Usage = PrettyPrintUsage(os.Stdout, f)

	return f
}

// Check if the FlagSet contains errors after parsing.
func (f *Flags) HasError() bool {
	return len(f.errors) > 0
}

// Return the underlying error map of the flag set.
func (f *Flags) Errors() ErrorMap {
	return f.errors
}

// Register a new command to the flag set.
//
// The defaultvalue cannot be nil!
//
// This is for type assertion purposes.
func (f *Flags) Register(name string, defaultValue any, description string, handler func(v Value) error) {
	var cmd *Command = &Command{
		Name:        name,
		Description: description,
		Default:     defaultValue,
		Handler:     handler,
	}
	f.Commands = append(f.Commands, cmd)
}

// Register a command for a pointer to a value.
//
// The defaultvalue cannot be nil!
//
// This is for type assertion purposes.
func (f *Flags) RegisterPtr(ptr, defaultValue any, name string, description string, handler func(v Value) error) {
	if ptr == nil {
		panic("ptr cannot be nil")
	}
	var unPtr = dePtr(ptr)
	if defaultValue == nil {
		defaultValue = newOf(false, unPtr)
	}
	if !typesEqual(unPtr, defaultValue) {
		panic("ptr and defaultValue must be of the same type")
	}
	if !isPtr(ptr) {
		panic("ptr must be a pointer")
	}
	var cmd *Command = &Command{
		Name:        name,
		Description: description,
		Default:     defaultValue,
		Handler:     handler,
		value:       ptr,
	}
	f.Commands = append(f.Commands, cmd)
}

// Register a new command to the flag set.
func (f *Flags) RegisterCommand(cmd *Command) {
	f.Commands = append(f.Commands, cmd)
}

// Execute the flag set.
func (f *Flags) Run() (wasRan bool) {
	for i := range f.Commands {
		f.Commands[i].Init(f.FlagSet)
	}
	f.FlagSet.Parse(os.Args[1:])
	var err error
	for i := range f.Commands {
		if shouldExecute(f, f.Commands[i].Name) ||
			!equalsNew(f.Commands[i].Default) ||
			isBool(f.Commands[i].Default) {

			if err = f.Commands[i].Execute(); err != nil {
				f.errors[f.Commands[i].Name] = err
			}

			wasRan = true || wasRan
		}
	}
	f.wasRan = true
	return wasRan
}

func (f *Flags) Ran() bool {
	return f.wasRan
}

// Reports if the flag was present in the arguments.
func shouldExecute(f *Flags, name string) bool {
	found := false
	f.FlagSet.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// Return a new value of the given type.
func newOf(ptr bool, v interface{}) interface{} {
	if ptr {
		return reflect.New(reflect.TypeOf(v)).Interface()
	}
	return reflect.Zero(reflect.TypeOf(v)).Interface()
}

// Check if the given value is equal to a new value of the same type.
func equalsNew(v interface{}) bool {
	return reflect.DeepEqual(v, newOf(false, v))
}

// Check if the given value is equal to a new value of the same type.
func typesEqual(a, b interface{}) bool {
	var typeOfA = reflect.TypeOf(a)
	var typeOfB = reflect.TypeOf(b)
	if typeOfA.Kind() == reflect.Ptr {
		typeOfA = typeOfA.Elem()
	}
	if typeOfB.Kind() == reflect.Ptr {
		typeOfB = typeOfB.Elem()
	}
	return typeOfA == typeOfB || typeOfA.Kind() == typeOfB.Kind()
}

// Check if the given value is a bool.
func isBool(v interface{}) bool {
	var b bool
	return typesEqual(v, b)
}

// Check if the given value is a string.
func isString(v any) bool {
	var b string
	return typesEqual(v, b)
}

// cast the pointer to the value, recursively.
func dePtr(v any) any {
	var typeOf = reflect.TypeOf(v)
	if typeOf.Kind() == reflect.Ptr {
		return reflect.ValueOf(v).Elem().Interface()
	}
	if typeOf.Kind() == reflect.Ptr {
		return dePtr(reflect.ValueOf(v).Elem().Interface())
	}
	return v
}

// isPtr checks if the given value is a pointer.
func isPtr(v any) bool {
	var typeOf = reflect.TypeOf(v)
	return typeOf.Kind() == reflect.Ptr
}
