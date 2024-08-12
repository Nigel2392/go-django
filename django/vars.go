package django

const (
	APPVAR_DEBUG         = "DEBUG"         // bool
	APPVAR_ALLOWED_HOSTS = "ALLOWED_HOSTS" // []string (e.g. ["*"])
	APPVAR_RECOVERER     = "RECOVERER"     // bool (use recoverer middleware)
	APPVAR_HOST          = "HOST"          // string
	APPVAR_PORT          = "PORT"          // string
	APPVAR_TLS_PORT      = "TLS_PORT"      // string
	APPVAR_TLS_CERT      = "TLS_CERT"      // /path/to/cert.pem
	APPVAR_TLS_KEY       = "TLS_KEY"       // /path/to/key.pem
	APPVAR_TLS_CONFIG    = "TLS_CONFIG"    // *tls.Config
	APPVAR_DATABASE      = "DATABASE"      // *sql.DB
)
