# Forms App Example

The `formsapp` example demonstrates how to handle HTML forms using the `go-django` form utilities.

## Overview

The forms application provides a comprehensive look at how to:
- Define form structures using the `forms` package.
- Initialize and populate forms with request data (`POST` or `GET`).
- Validate user input automatically using form validators.
- Render form errors and fields cleanly using templates and widgets.

## Key Concepts

`go-django` forms are highly customizable. You can use widgets to specify how form fields are rendered in HTML, add custom validation logic, and process valid form submissions within your views or generic view classes like `FormView`.

## Running the Example

1. Navigate to the `examples/formsapp` directory.
2. Run `go run main.go`.
3. Open your browser to test form validation and submission.
