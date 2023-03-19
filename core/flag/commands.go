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

// Initialize the command, this will add the command to the std.FlagSet.
func (c *Command) Init(f *flag.FlagSet) {
	if c.Default == nil {
		panic(fmt.Sprintf("Default value for %s is nil", c.Name))
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
