# Permissions

The `github.com/Nigel2392/go-django/src/permissions` package provides a pluggable permission-checking facade for HTTP requests. Templates, views, and contrib apps call `permissions.HasPermission` and `permissions.HasObjectPermission`, which delegate to a configured `PermissionTester` backend.

Do note that package-level variables such as `Tester` and `LOG_IF_NONE_FOUND` should be configured once in your main application startup, rather than mutated directly inside reusable third-party apps.

---

## Core Interface and Variables

```go
package permissions

type PermissionTester interface {
	HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool
	HasPermission(r *http.Request, perms ...string) bool
}

var (
	Tester PermissionTester

	LOG_IF_NONE_FOUND      = true
	DEFAULT_IF_NONE_FOUND  = defaultLog
	DEFAULT_HAS_PERMISSION = false
)
```

- **`PermissionTester`**: Defines the contract for request-scoped and object-scoped permission checks. If your application only uses global permissions, `HasObjectPermission` can simply delegate to `HasPermission(r, perms...)`.
- **`Tester`**: Global `PermissionTester` instance used by `HasPermission` and `HasObjectPermission`.
- **`LOG_IF_NONE_FOUND`**: Controls whether `defaultLog` emits a warning log when `Tester` is `nil`. Defaults to `true`.
- **`DEFAULT_IF_NONE_FOUND`**: Fallback function invoked when `Tester` is `nil`. Defaults to `defaultLog`.
- **`DEFAULT_HAS_PERMISSION`**: Result returned by `defaultLog` for non-admin users when no `Tester` is configured. Defaults to `false` (deny by default).

---

## Functions

```go
func HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool
func HasPermission(r *http.Request, perms ...string) bool
```

Both functions evaluate checks using the following rules:

1. If `len(perms) == 0`, they immediately return `true`.
2. If `Tester == nil`, they call `DEFAULT_IF_NONE_FOUND(r, perms...)`.
3. Otherwise, they delegate to `Tester.HasObjectPermission(r, obj, perms...)` or `Tester.HasPermission(r, perms...)`.

### Default Behavior (`defaultLog`)

When `Tester` is `nil`, the default `DEFAULT_IF_NONE_FOUND` function (`defaultLog`) checks the authenticated user attached to the request:

```go
func defaultLog(r *http.Request, perms ...string) bool {
	var user = authentication.Retrieve(r)
	if user != nil && user.IsAdmin() {
		return true
	}

	if LOG_IF_NONE_FOUND {
		logger.Warnf(
			"No permission testers found for \"%s\" (%s)",
			strings.Join(perms, "\", \""),
			r.URL.Path,
		)
	}

	return DEFAULT_HAS_PERMISSION
}
```

- If `user != nil && user.IsAdmin()`, `defaultLog` **grants permission (`true`) immediately without logging a warning**.
- For all other requests (anonymous or non-admin users), `defaultLog` logs a warning if `LOG_IF_NONE_FOUND` is `true` and returns `DEFAULT_HAS_PERMISSION` (`false`, denying access by default).

---

## Implementing a Custom `PermissionTester`

To enable permission checks across views and templates, implement `PermissionTester` and assign it to `permissions.Tester`:

```go
package main

import (
	"net/http"
	"slices"

	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
)

type RoleUser interface {
	authentication.User
	Roles() []string
}

type RolePermissions struct{}

func (p *RolePermissions) HasPermission(r *http.Request, perms ...string) bool {
	user := authentication.Retrieve(r)
	if user == nil || !user.IsAuthenticated() {
		return false
	}
	if user.IsAdmin() {
		return true
	}

	roleUser, ok := user.(RoleUser)
	if !ok {
		return false
	}

	roles := roleUser.Roles()
	for _, perm := range perms {
		if !slices.Contains(roles, perm) {
			return false
		}
	}
	return true
}

func (p *RolePermissions) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool {
	return p.HasPermission(r, perms...)
}

func main() {
	permissions.Tester = &RolePermissions{}
}
```

### Using `contrib/auth`

If you use Go-Django's built-in authentication app (`contrib/auth`), `contrib/auth/users` provides `users.NewPermissionsBackend()` which implements `permissions.PermissionTester` with user-level and group-level permission checks. See the [auth permissions documentation](./apps/auth/permissions.md) for setup details.
