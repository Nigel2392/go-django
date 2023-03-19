package flag

import (
	"flag"
	"os"
	"reflect"
)

type Allowed interface {
	string | bool | int64 | int | uint64 | uint | float64
}

type Flags struct {
	Commands []Command
	errors   ErrorMap
	FlagSet  *flag.FlagSet
	Info     string
}

type ErrorHandling int

const (
	ContinueOnError = ErrorHandling(flag.ContinueOnError)
	ExitOnError     = ErrorHandling(flag.ExitOnError)
	PanicOnError    = ErrorHandling(flag.PanicOnError)
)

func NewFlags(name string, handling ...ErrorHandling) *Flags {
	var errHandler ErrorHandling = ContinueOnError
	if len(handling) > 0 {
		errHandler = handling[0]
	}

	var f = &Flags{
		Commands: make([]Command, 0),
		errors:   make(ErrorMap, 0),
		FlagSet:  flag.NewFlagSet(name, flag.ErrorHandling(errHandler)),
	}

	f.FlagSet.Usage = PrettyPrintUsage(os.Stdout, f)

	return f
}

func (f *Flags) HasError() bool {
	return len(f.errors) > 0
}

func (f *Flags) Errors() ErrorMap {
	return f.errors
}

func (f *Flags) Register(name string, defaultValue any, description string, handler func(v Value) error) {
	var cmd Command = Command{
		Name:        name,
		Description: description,
		Default:     defaultValue,
		Handler:     handler,
	}
	f.Commands = append(f.Commands, cmd)
}

func (f *Flags) RegisterCommand(cmd Command) {
	f.Commands = append(f.Commands, cmd)
}

func (f *Flags) Run() (wasRan bool) {
	for i := range f.Commands {
		f.Commands[i].Init(f.FlagSet)
	}
	f.FlagSet.Parse(os.Args[1:])
	var err error
	for i := range f.Commands {
		if (f.shouldExecute(f.Commands[i].Name) || equalsNew(f.Commands[i].value)) ||
			isBool(f.Commands[i].Default) {

			if err = f.Commands[i].Execute(); err != nil {
				f.errors[f.Commands[i].Name] = err
			}

			wasRan = true || wasRan
		}
	}
	return
}

func newOf(ptr bool, v interface{}) interface{} {
	if ptr {
		return reflect.New(reflect.TypeOf(v)).Interface()
	}
	return reflect.Zero(reflect.TypeOf(v)).Interface()
}

func equalsNew(v interface{}) bool {
	return reflect.DeepEqual(v, newOf(false, v))
}

func (f *Flags) shouldExecute(name string) bool {
	found := false
	f.FlagSet.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func typesEqual(a, b interface{}) bool {
	var typeOfA = reflect.TypeOf(a)
	var typeOfB = reflect.TypeOf(b)
	if typeOfA.Kind() == reflect.Ptr {
		typeOfA = typeOfA.Elem()
	}
	if typeOfB.Kind() == reflect.Ptr {
		typeOfB = typeOfB.Elem()
	}
	return typeOfA == typeOfB
}

func isBool(v interface{}) bool {
	var b bool
	return typesEqual(v, b)
}

func isString(v any) bool {
	var b string
	return typesEqual(v, b)
}

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
