# Server Errors Example

The `server_errors` example demonstrates how to handle custom server errors using the `go-django` framework.

## Key Concepts

- **Custom Error Handlers**: Define custom HTTP handler functions for specific HTTP status codes (e.g., 400, 403, 404, 500).
- **Configuration**: Use `django.APPVAR_ErrorCode(http.StatusNotFound)` to map error codes to handler functions directly in the app configuration.
- **Testing Errors**: The example provides routes like `/raise/<<code>>` to simulate and test specific HTTP error responses effortlessly.

## Running the Example

1. Run the application from `examples/server_errors` using `go run main.go`.
2. Visit `http://127.0.0.1:8080/raise/404` to see the custom 404 handler.
3. Visit `http://127.0.0.1:8080/raise/500` to simulate an internal server error.
4. Replace the code in the URL with any valid HTTP status code to test its handler.
