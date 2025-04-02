package django

const (

	// Run the application in debug mode
	APPVAR_DEBUG = "DEBUG" // bool

	// Allowed hosts for the application, e.g. ["*"] - if the host is not in this list, the application will return an error when serving.
	APPVAR_ALLOWED_HOSTS = "ALLOWED_HOSTS" // []string (e.g. ["*"])

	// Use recoverer middleware for the application to recover from panics
	APPVAR_RECOVERER = "RECOVERER" // bool (use recoverer middleware)

	// Host address to bind the application's server to, e.g. "localhost"
	APPVAR_HOST = "HOST" // string

	// Port to bind the application's server to, e.g. "8080"
	APPVAR_PORT = "PORT" // string

	// The URL to serve static files from, e.g. "/static/"
	APPVAR_STATIC_URL = "STATIC_URL" // string

	// The port to bind the application's server to for TLS, e.g. "443"
	APPVAR_TLS_PORT = "TLS_PORT" // string

	// The path to the certificate file for TLS
	APPVAR_TLS_CERT = "TLS_CERT" // /path/to/cert.pem

	// The path to the key file for TLS
	APPVAR_TLS_KEY = "TLS_KEY" // /path/to/key.pem

	// A custom TLS configuration for the application
	APPVAR_TLS_CONFIG = "TLS_CONFIG" // *tls.Config

	// The database connection for the application
	APPVAR_DATABASE = "DATABASE" // *sql.DB

	// Continue running the application after executing cli- commands
	APPVAR_CONTINUE_AFTER_COMMANDS = "CONTINUE_AFTER_COMMAND" // bool

	// Log all routes that are accessed
	APPVAR_ROUTE_LOGGING_ENABLED = "ROUTE_LOGGING_ENABLED" // bool

	// Wether the webserver is behind a proxy
	// This is so the application knows to use different headers to, for example
	// get the remote address of the client
	APPVAR_REQUESTS_PROXIED = "REQUESTS_PROXIED" // bool

	// The session manager for the application
	APPVAR_SESSION_MANAGER = "SESSION_MANAGER" // *scs.SessionManager

)
