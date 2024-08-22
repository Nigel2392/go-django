# Hooks

Hooks are a way to run code at specific points in project's code. They are similar to [Wagtail's hook system](https://docs.wagtail.org/en/stable/reference/hooks.html); with one key difference.

They are typically used to allow third-party apps to modify the behavior of the project without needing to modify the project's code directly.

When working with models and other structured data, we prefer working with [signals](https://github.com/Nigel2392/go-signals) instead of hooks.

Signals not only provide an extra level of type- safety, but also allow for a simpler way to connect / disconnect.

When defining or using hooks, it is generally a good idea to have provided good documentation on what the hook does, it's name, what the expected return value is, as well as the function type.

We recommend to do most of your definitions in a `hooks.go` file to easily determine which apps use hooks and which hooks are available.
