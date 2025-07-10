package auth

const (
	APPVAR_AUTH_EMAIL_LOGIN   = "AUTH_EMAIL_LOGIN"   // type: bool
	APPVAR_REGISTER_AUTH_URLS = "REGISTER_AUTH_URLS" // type: bool
	APPVAR_LOGIN_REDIRECT_URL = "LOGIN_REDIRECT_URL" // type: string || func(*http.Request) string

	DEFAULT_LOGIN_REDIRECT_URL = "/" // default value for LOGIN_REDIRECT_URL
)
