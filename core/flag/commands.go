package flag

import (
	"flag"
	"fmt"
	"reflect"
)

// A command to add to a flag set.
//
// This command will be parsed and handled by the flag set.
//
// The default value needs to be set!
//
// This is for type assertion of the flag.
//
// Available types are:
//   - string
//   - bool
//   - int64
//   - int
//   - uint64
//   - uint
//   - float64
type Command struct {
	Name        string
	Description string
	Default     any
	Handler     func(v Value) error

	value any
}

func PtrCommand[T Allowed](name string, description string, ptr *T, def T, handler func(v Value) error) *Command {
	if ptr == nil {
		panic("Pointer is nil")
	}
	return &Command{
		Name:        name,
		Description: description,
		Default:     def,
		Handler:     handler,
		value:       ptr,
	}
}

// Initialize the command, this will add the command to the std.FlagSet.
func (c *Command) Init(f *flag.FlagSet) {
	if c.Default == nil {
		panic(fmt.Sprintf("Default value for %s is nil", c.Name))
	}

	if c.value != nil {
		var unPtr = dePtr(c.value)
		if !typesEqual(unPtr, c.Default) {
			panic(fmt.Sprintf("Default value type does not match the pointer type: %T != %T", unPtr, c.Default))
		}
		switch c.value.(type) {
		case *string:
			f.StringVar(c.value.(*string), c.Name, c.Default.(string), c.Description)
		case *bool:
			f.BoolVar(c.value.(*bool), c.Name, c.Default.(bool), c.Description)
		case *int64:
			f.Int64Var(c.value.(*int64), c.Name, c.Default.(int64), c.Description)
		case *int:
			f.IntVar(c.value.(*int), c.Name, c.Default.(int), c.Description)
		case *uint64:
			f.Uint64Var(c.value.(*uint64), c.Name, c.Default.(uint64), c.Description)
		case *uint:
			f.UintVar(c.value.(*uint), c.Name, c.Default.(uint), c.Description)
		case *float64:
			f.Float64Var(c.value.(*float64), c.Name, c.Default.(float64), c.Description)
		default:
			panic(fmt.Sprintf("Unsupported type: %T", c.value))
		}
		return
	}
	var typeOf = reflect.TypeOf(c.Default)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	switch typeOf.Kind() {
	case reflect.String:
		c.value = f.String(c.Name, any(c.Default).(string), c.Description)
	case reflect.Bool:
		c.value = f.Bool(c.Name, any(c.Default).(bool), c.Description)
	case reflect.Int64:
		c.value = f.Int64(c.Name, any(c.Default).(int64), c.Description)
	case reflect.Int:
		c.value = f.Int(c.Name, any(c.Default).(int), c.Description)
	case reflect.Uint64:
		c.value = f.Uint64(c.Name, any(c.Default).(uint64), c.Description)
	case reflect.Uint:
		c.value = f.Uint(c.Name, any(c.Default).(uint), c.Description)
	case reflect.Float64:
		c.value = f.Float64(c.Name, any(c.Default).(float64), c.Description)
	default:
		panic(fmt.Sprintf("Unsupported type: %s", typeOf.Kind()))
	}
}

// Execute the command, this will call the handler function
// with the value of the parsed command.
func (c *Command) Execute() error {
	if c.Handler != nil {
		return c.Handler(&value{dePtr(c.value)})
	}
	return nil
}
