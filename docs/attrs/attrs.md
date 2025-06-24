# `attrs` Package Documentation

The `attrs` package enables defining and managing model attributes with support for auto-generated forms, field validation, and custom form widgets.

Table of Contents:

- [Functions](./functions.md)
- [Interfaces](./interfaces.md)
- [Implementation](./implementation.md)
- [Model Meta](./model-meta.md)
- [Hooks and Signals](./hooks_signals.md)

It also allows for defining relations between models - for example [go-django-queries](https://github.com/Nigel2392/go-django/queries)  
uses this to define relations between models and overall manage the database schema.

When models are registered in an apps' `ModelObjects` attribute (`Model()` method), go-django will automatically register all your models  
to the `attrs` package with the `attrs.RegisterModel` function.

Once the models are registered, a global registry object will get updated with said model, and it's ['static' definition of fields](./model-meta.md).

This allows for easy access to the fields of a model, **if you only need the attributes** and definition of the fields.
