# OpenAuth2 App Example

The `openauth2app` example demonstrates how to integrate OAuth2 authentication into a `go-django` application.

## Key Features

- **Provider Integration**: Uses `golang.org/x/oauth2` to integrate providers like Google.
- **App Configuration**: Configures the `openauth2.AppConfig` with client IDs and secrets via `.env`.
- **Data Mapping**: Maps OAuth2 user data (e.g., Google's `GoogleUser`) to application-specific structures.
- **Custom Redirects**: Demonstrates defining the `RedirectAfterLogin` and `RedirectAfterLogout` behaviors.
- **Media and DB**: Leverages a SQLite database and an `fs` media backend for storing user data and pictures.

## Usage

1. Create a `.env` file inside `.private/.env` with your `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`.
2. Navigate to `examples/openauth2app` and run `go run main.go`.
3. Access the application on `http://127.0.0.1:8080` and authenticate using your Google account.
