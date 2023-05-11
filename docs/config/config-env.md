# An example of a fully implemented environment file

This will also include extra explanation to the environment variables.

#### SECURITY WARNING: keep the secret key used in production secret!

```
SECRET_KEY="" # The secret key used by the application, and authentication.
```

#### Allowed hosts.

```
ALLOWED_HOSTS="127.0.0.1" "localhost" # If a request.Host is not in this list, it will error.
```

#### Address to host the server on.

```
HOST="127.0.0.1" # The interface to listen on.

PORT=8080 # The port to listen on.
```

#### SSL Certificate and Key file.

```
SSL_CERT_FILE=None # Or a path to a certificate.

SSL_KEY_FILE=None # A path to a private key file.
```

#### Requests per second to allow.

```
REQUESTS_PER_SECOND=10 # Maximum allowed of requests per second. Will error if exceeded.

REQUEST_BURST_MULTIPLIER=3 # Maximum burst multiplier for the requests.
```

#### Database settings.

**If none DisableAuthentication must be set to true in the config.**

```
DB_NAME=None # The name of the database.

DB_USER=None # The username to use for the database.

DB_PASS=None # The password to use for the database.

DB_HOST=None # The host to connect to.

DB_PORT=None # The port to connect to.

DB_DSN_PARAMS=None # DSN query parameters after the ? in the DSN string.
```

#### Template settings.

```
# The directory to make a os.DirFS from.
TEMPLATE_DIR="assets/templates"

# The suffixes of template files.
TEMPLATE_BASE_SUFFIXES=".html", ".htm", ".xml", "tmpl", "tpl"

# The base template directory
TEMPLATE_BASE_DIRS="base"

# The directory to look for templates in.
TEMPLATE_DIRS="templates"

# Use a cache to fetch the templates after they have been loaded once.
TEMPLATE_CACHE=False
```

#### Email settings.

```
# The email host to use.
EMAIL_HOST = "smtp.gmail.com"

# The port to use.
EMAIL_PORT = 465

# The username to use.
EMAIL_USERNAME = "test@gmail.com"

# The password to use.
EMAIL_PASSWORD = "password"

# Use TLS.
EMAIL_USE_TLS = False

# Use SSL.
EMAIL_USE_SSL = True

# The email address to use as the from address.
EMAIL_FROM = $EMAIL_USERNAME
```

#### Staticfiles/mediafiles options

```
STATIC_DIR="assets/static" # The directory where staticfiles are located.
```

#### Lifetime of the session cookie.

Format: 1w3d12h (1 week, 3 days, 12 hours)

3d12h (3 days, 12 hours)

12h (12 hours)

1w (1 week)

```
# The lifetime of the session cookie.
# Format: 1w3d12h (1 week, 3 days, 12 hours)
SESSION_COOKIE_LIFETIME="1w3d12h" 

# Name to set for the session cookie. If not set, the default name is used.
SESSION_COOKIE_NAME="sessionid"

# Idle timeout for the cookie
SESSION_IDLE_TIMEOUT=None

# Domain to set for the session cookie. If not set, the default domain is used.
SESSION_COOKIE_DOMAIN=None

# Path to set for the session cookie. If not set, the default path is used.
SESSION_COOKIE_PATH="/"

# SESSION_COOKIE_SAMESITE options are:
#   1. SameSiteDefaultMode
#   2. SameSiteLaxMode
#   3. SameSiteStrictMode
#   4. SameSiteNoneMode
SESSION_COOKIE_SAME_SITE=2

// Set to true to only allow HTTPS connections.
SESSION_COOKIE_SECURE=False

// Set to true to only allow HTTP connections.
SESSION_COOKIE_HTTP_ONLY=True

// Set to true to allow the session cookie to persist after the browser is closed.
SESSION_COOKIE_PERSIST=True
```