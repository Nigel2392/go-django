# Hooks and Signals for the `attrs` package

Hooks and signals provide a way to hook or callback into the `attrs` package.

These include:

- Hooking into the model registration process
- Hooking into the model's form field generation process
- Hooking into the model's default value generation process

## Hooks

Hooks are functions that are called before or after a certain action.

They can be used to add custom logic before or after a certain action.

Custom helper functions are provided to register hooks for use in the `attrs` package.

### Form Field Hooking

An example of a form field hook is the `RegisterFormFieldType` function.

This function is used to register which form field to use for a given value of type.

Register form fields for custom types:

```go
RegisterFormFieldType(
  json.RawMessage{},
  func(opts ...func(fields.Field)) fields.Field {
    return fields.JSONField[json.RawMessage](opts...)
  },
)
```

### Default Value Hooking

An example of a default value hook is the `RegisterDefaultType` function.

This function is used to register which default value to use for a given value of type.

It uses passes reflect values to the hook function, and expects a value of type `any` to be returned.

Register default values for custom types:

```go
RegisterDefaultType(
  json.RawMessage{},
  func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool) {
    if field_v.IsValid() && field_v.Type() == reflect.TypeOf(json.RawMessage{}) {
      return json.RawMessage{}, true
    }
    return nil, false
  },
)
```

## Signals

Signals are used to send messages to other parts of the application.

They can be used in the `attrs` package to hook into the model registration process.

This should preferrably be done inside an `init()` function, or on the global package level.

Example usage:

```go
func init() {
    attrs.OnBeforeModelRegister.Listen(func(s signals.Signal[attrs.Definer], obj attrs.Definer) error {
        // Do something before the model is registered
        return nil
    })
    attrs.OnModelRegister.Listen(func(s signals.Signal[attrs.Definer], obj attrs.Definer) error {
        // Do something after the model is registered
        return nil
    })
}
```
