# Hooks

Hooks are a way to run code at specific points in project's code. They are similar to [Wagtail's hook system](https://docs.wagtail.org/en/stable/reference/hooks.html); with one key difference.

They are typically used to allow third-party apps to modify the behavior of the project without needing to modify the project's code directly.

When working with models and other structured data, we prefer working with [signals](https://github.com/Nigel2392/go-signals) instead of hooks.

Signals not only provide an extra level of type- safety, but also allow for a simpler way to connect / disconnect.

When defining or using hooks, it is generally a good idea to have provided good documentation on what the hook does, it's name, and the hook's function type signature.

We recommend to do most of your definitions in a `hooks.go` file to easily determine which apps use hooks and which hooks are available.

For more information on how hooks work - see the [hooks package](https://github.com/Nigel2392/goldcrest).

## Django Defined Hooks

### django.ServerError

*Type Signature: `django.ServerErrorHook`*

This hook is called when a server error occurs.

It is called after the error has been logged and before a response is sent to the client.

The hook is passed the request, response, the error and the web-app instance.

```go
goldcrest.Register(
	django.HOOK_SERVER_ERROR, 0,
	func(w http.ResponseWriter, r *http.Request, app *django.Application, serverError except.ServerError) {
        // Do something with the error, if nothing is written to the response, the default error page is shown
    },
)
```

### attrs.FormFieldForType

*Type Signature: `attrs.FormFieldGetter`*

Registering a form field for a type allows the form generator to automatically generate a form field for the type.

I.E. for a `json.RawMessage` type, we automatically generate a `textarea` field.

Must return a `fields.Field` and a boolean indicating if the field returned is valid.

```go
goldcrest.Register(
    attrs.HookFormFieldForType, 0,
    func(f Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
		if field_v.IsValid() && field_v.Type() == typ || new_field_t_indirected == typ {
			return fields.JSONField[typ](opts...), true
		}
		return nil, false
	}
)
```

### attrs.DefaultForType

*Type Signature: `attrs.DefaultGetter`*

Registering a default for a type allows the form generator to automatically generate a default value for the type.

This might be nescessary for types that must initialize with some properties set.

Example:

```go
type MyBlock struct {
    Text           string
    wasInitialized bool
}

var myBlockTyp = reflect.TypeOf(MyBlock{})

goldcrest.Register(
	attrs.DefaultForType, 0,
	attrs.DefaultGetter(func(f attrs.Field, t reflect.Type, v reflect.Value) (any, bool) {
		if v.Type().Implements(blockTyp) { // equivalent to if t == myBlockTyp
            return &MyBlock{wasInitialized: true}, true
		}
		return nil, false
	}),
)
```
