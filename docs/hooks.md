# Hooks and Signals

Go-Django provides two extension mechanisms for decoupling behavior across apps:

1. **Hooks** (`github.com/Nigel2392/goldcrest`): Function registries keyed by string name and integer priority order, ideal for overriding behavior, customizing handlers, or registering global adapters.
2. **Signals** (`github.com/Nigel2392/go-signals`): Type-safe pub/sub channels (`signals.Signal[T]`) for lifecycle events, HTTP request interception, model registration, and authentication events.

---

## Hooks (`goldcrest`)

Register hooks with `goldcrest.Register(identifier, order, fn...)`:

- **`order` semantics:** Hooks are sorted in **ascending** `order`—lower numbers run earlier and higher numbers run later. Hooks registered with the same `order` execute in registration order.
- Depending on the hook site, Go-Django either executes **all** registered hooks or **short-circuits** on the first hook that handles the call. Each core hook's execution rule is noted below.

---

## Core Hooks Reference

### `django.HOOK_SETUP_NOSURF` (`"django.SetupNosurf"`)

*Signature:* `django.NosurfSetupHook` (`func(app *django.Application, handler *nosurf.CSRFHandler)`)

Runs inside `app.Serve()` when building the `nosurf` CSRF middleware (when `APPVAR_DISABLE_NOSURF` is `false`). All registered hooks execute in order. Note that there is no `*http.Request` argument in this hook signature.

```go
goldcrest.Register(
    django.HOOK_SETUP_NOSURF, 0,
    django.NosurfSetupHook(func(app *django.Application, csrfMiddleware *nosurf.CSRFHandler) {
        csrfMiddleware.ExemptPath("/webhooks/stripe")
    }),
)
```

### `django.HOOK_SERVER_STARTUP` (`"django.ServerStart"`)

*Signature:* `django.DjangoHook` (`func(*django.Application) error`)

Runs inside `app.Serve()` right before the HTTP and/or HTTPS servers begin listening. Hooks run in order and `Serve()` aborts immediately if any hook returns a non-nil error.

```go
goldcrest.Register(
    django.HOOK_SERVER_STARTUP, 0,
    django.DjangoHook(func(app *django.Application) error {
        // Run startup setup before listening
        return nil
    }),
)
```

### `django.HOOK_SERVER_SHUTDOWN` (`"django.ServerQuit"`)

*Signature:* `django.DjangoHook` (`func(*django.Application) error`)

Runs inside `app.Quit()` after shutting down the HTTP/HTTPS servers. All registered hooks run and any returned errors are joined via `errors.Join`.

```go
goldcrest.Register(
    django.HOOK_SERVER_SHUTDOWN, 0,
    django.DjangoHook(func(app *django.Application) error {
        // Clean up background resources
        return nil
    }),
)
```

### `django.HOOK_SERVER_ERROR` (`"django.ServerError"`)

*Signature:* `django.ServerErrorHook` (`func(w http.ResponseWriter, r *http.Request, app *django.Application, err except.ServerError)`)

Runs in `app.ServerError` after logging an `except.ServerError`, unless a custom `APPVAR_ErrorCode(code)` handler is configured in settings. All registered hooks execute using a `markedResponseWriter`; if no hook writes headers or body bytes, Go-Django falls back to `http.Error(w, message, code)`.

```go
goldcrest.Register(
    django.HOOK_SERVER_ERROR, 0,
    django.ServerErrorHook(func(w http.ResponseWriter, r *http.Request, app *django.Application, err except.ServerError) {
        w.WriteHeader(err.StatusCode())
        w.Write([]byte(err.UserMessage()))
    }),
)
```

### `models.MODEL_SAVE_HOOK` (`"django.Model.Save"`)

*Signature:* `models.ModelFunc` (`func(c context.Context, m attrs.Definer) (changed bool, err error)`)

Called by `models.SaveModel` when `m` does not implement `models.ContextSaver` or `models.Saver`. Hooks execute in order and **stop immediately** on the first hook that returns `changed == true` or a non-nil `err`.

```go
goldcrest.Register(
    models.MODEL_SAVE_HOOK, 0,
    models.ModelFunc(func(c context.Context, m attrs.Definer) (bool, error) {
        // Custom persistence backend logic
        return false, nil
    }),
)
```

### `models.MODEL_DELETE_HOOK` (`"django.Model.Delete"`)

*Signature:* `models.ModelFunc` (`func(c context.Context, m attrs.Definer) (changed bool, err error)`)

Called by `models.DeleteModel` when `m` does not implement `models.ContextDeleter` or `models.Deleter`. Hooks execute in order and **stop immediately** on the first hook that returns `changed == true` or a non-nil `err`.

### `attrs.HookFormFieldForType` (`"attrs.FormFieldForType"`)

*Signature:* `attrs.FormFieldGetter` (`func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)`)

Consulted by `FieldDef.FormField()` when no explicit field-level `FormField` generator is configured. **Stops on the first hook returning `(field, true)`.**

For common type mappings, `attrs.RegisterFormFieldType(valueOfType, getField)` and `attrs.RegisterFormFieldGetter(valueOfType, getField)` provide convenient wrappers (registered at order `50`).

```go
var myType = reflect.TypeOf(MyCustomJSON{})

goldcrest.Register(
    attrs.HookFormFieldForType, 0,
    attrs.FormFieldGetter(func(f attrs.Field, t reflect.Type, v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
        if (v.IsValid() && v.Type() == myType) || t == myType {
            return fields.JSONField[MyCustomJSON](opts...), true
        }
        return nil, false
    }),
)
```

### `attrs.DefaultForType` (`"attrs.DefaultForType"`)

*Signature:* `attrs.DefaultGetter` (`func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value) (interface{}, bool)`)

Consulted by `FieldDef.GetDefault()` to produce initialized default values for Go types. **Stops on the first hook returning `(val, true)`.** You can also use `attrs.RegisterDefaultType(valueOfType, getDefault)` (registered at order `100`).

```go
type MyBlock struct {
    Text           string
    wasInitialized bool
}

var myBlockTyp = reflect.TypeOf(MyBlock{})

goldcrest.Register(
    attrs.DefaultForType, 0,
    attrs.DefaultGetter(func(f attrs.Field, t reflect.Type, v reflect.Value) (any, bool) {
        if (v.IsValid() && v.Type() == myBlockTyp) || t == myBlockTyp {
            return &MyBlock{wasInitialized: true}, true
        }
        return nil, false
    }),
)
```

### `forms.FormTemplateFSHookName` and `forms.FormTemplateStaticHookName`

*Signatures:*
- `forms.FormTemplateFSHook` (`func(fSys fs.FS, cnf *tpl.Config) fs.FS`) for `"forms.TemplateFSHook"`
- `forms.FormTemplateStaticHook` (`func(fSys fs.FS) fs.FS`) for `"forms.TemplateStaticHook"`

Invoked once on `tpl.FirstRender()` when `forms` initializes its widget templates and static files. All registered hooks execute in order, chaining the returned `fs.FS` from each hook into the next so third-party apps can override form widget templates or assets.

---

## Core Signals

Go-Django emits typed `signals.Signal[T]` events across startup, HTTP requests, model registration, and authentication. For model-level `attrs` signals, see [Attrs Hooks and Signals](./attrs/hooks_signals.md).

### Application Lifecycle Signals (`src/core/core_signals.go`)

All three lifecycle signals have type `signals.Signal[any]` and pass `*django.Application` as the payload during `app.Initialize()`:

- `core.BeforeModelsReady` (`"django.BeforeModelsReady"`): Sent after `dbtype.Lock()` and immediately before app models are registered with `djangoapps`, `contenttypes`, and `attrs`.
- `core.OnModelsReady` (`"django.OnModelsReady"`): Sent after all models are registered, right before `attrs.ResetDefinitions` runs.
- `core.OnDjangoReady` (`"django.OnReady"`): Sent at the very end of `app.Initialize()`, after all apps' `OnReady()` callbacks complete and `a.initialized` is set to `true`.

```go
core.OnDjangoReady.Listen(context.Background(), func(ctx context.Context, s signals.Signal[any], app any) error {
    a := app.(*django.Application)
    a.Log.Info("Application is ready")
    return nil
})
```

### HTTP Request Signals (`src/core/http_signals.go`)

`django.RequestSignalMiddleware` emits `*core.HttpSignal` before and after `a.Mux` serves a request:

```go
type HttpSignal struct {
    W http.ResponseWriter
    R *http.Request
    H http.Handler
}
```

- `core.SIGNAL_BEFORE_REQUEST` (`"http.before_request"`): Emitted before calling `signal.H.ServeHTTP(signal.W, signal.R)`. Returning `django.ErrServeCanceled` cancels normal routing immediately so your listener can write a custom response and skip the downstream handler and `SIGNAL_AFTER_REQUEST`.
- `core.SIGNAL_AFTER_REQUEST` (`"http.after_request"`): Emitted after `signal.H.ServeHTTP` returns (unless canceled with `django.ErrServeCanceled`).

```go
core.SIGNAL_BEFORE_REQUEST.Listen(context.Background(), func(ctx context.Context, s signals.Signal[*core.HttpSignal], hs *core.HttpSignal) error {
    if hs.R.URL.Path == "/maintenance" {
        hs.W.WriteHeader(http.StatusServiceUnavailable)
        hs.W.Write([]byte("Maintenance in progress"))
        return django.ErrServeCanceled
    }
    return nil
})
```

### Authentication Signals (`src/core/auth_signals.go`)

Authentication apps emit these core signals (`github.com/Nigel2392/go-django/src/core`; see also [Auth App](./apps/auth/app.md)):

- `core.SIGNAL_BEFORE_USER_CREATE`, `core.SIGNAL_AFTER_USER_CREATE` (`signals.Signal[core.User]`)
- `core.SIGNAL_BEFORE_USER_UPDATE`, `core.SIGNAL_AFTER_USER_UPDATE` (`signals.Signal[core.User]`)
- `core.SIGNAL_BEFORE_USER_DELETE`, `core.SIGNAL_AFTER_USER_DELETE` (`signals.Signal[any]`, payload is the user primary key)
- `core.SIGNAL_USER_LOGGED_IN`, `core.SIGNAL_USER_LOGGED_OUT` (`signals.Signal[core.UserWithRequest]`)
- `core.SIGNAL_LOGIN_FAILED` (`signals.Signal[*http.Request]`)

> [!NOTE]
> Callers of the authentication signals ignore returned errors from listeners.
