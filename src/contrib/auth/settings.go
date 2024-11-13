package auth

const (
	APP_AUTH_EMAIL_LOGIN   = "AUTH_EMAIL_LOGIN"   // type: bool
	APP_REGISTER_AUTH_URLS = "REGISTER_AUTH_URLS" // type: bool
	APP_LOGIN_REDIRECT_URL = "LOGIN_REDIRECT_URL" // type: string || func(*http.Request) string
)
