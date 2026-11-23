# Permissions Backend and RBAC (`contrib/auth/users`)

Go-Django's core [Permissions](../../permissions.md) package delegates permission checks to the global `permissions.Tester` variable. The `contrib/auth/users` package provides `PermissionsBackend`, an RBAC implementation backed by `users.Permission`, `users.Group`, and the `users.Base` model mixin.

For authentication setup and user models, see:

- [Authentication App (`auth`)](./app.md)
- [User Model, Passwords, and Custom Users](./user.md)

---

## Wiring `permissions.Tester`

To enable `users.PermissionsBackend`, assign it to `permissions.Tester` during application startup:

```go
import (
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/permissions"
)

func main() {
	permissions.Tester = users.NewPermissionsBackend()
	// ...
}
```

### Why `auth.Init` Does Not Set `permissions.Tester` Automatically

Modular apps in Go-Django intentionally avoid overwriting global package-level hooks like `permissions.Tester`. Leaving `permissions.Tester` explicit in `main` ensures that:

- Registering `auth` alongside another user backend (such as [OAuth2](../oauth2.md)) or a custom `PermissionTester` never silently clobbers your chosen backend based on `django.Apps(...)` ordering.
- Applications that only use `IsAdmin()` or custom permission rules remain unaffected.

When `permissions.Tester` is `nil`, `permissions.HasPermission` and `permissions.HasObjectPermission` fall back to `permissions.DEFAULT_IF_NONE_FOUND`, which grants access to `user.IsAdmin()` and logs a warning while returning `permissions.DEFAULT_HAS_PERMISSION` (`false`) for all non-admin users.

---

## Evaluation Order and Caching

`users.NewPermissionsBackend()` implements `permissions.PermissionTester`:

```go
type PermissionsBackend struct{}

func NewPermissionsBackend() *PermissionsBackend
func (pb *PermissionsBackend) HasPermission(r *http.Request, perms ...string) bool
func (pb *PermissionsBackend) HasObjectPermission(r *http.Request, obj interface{}, perms ...string) bool
```

Calling `permissions.HasPermission(r, perms...)` or `permissions.HasObjectPermission(r, obj, perms...)` evaluates checks in the following order:

1. **Request user lookup**: Calls `authentication.Retrieve(r)`. If `nil`, returns `false`.
2. **Empty permissions slice**: If `len(perms) == 0`, returns `true`.
3. **Authentication gate**: Calls `user.IsAuthenticated()`. If `false` (including an inactive administrator where `IsActive == false`), immediately returns `false`.
4. **Administrator bypass**: Calls `user.IsAdmin()` (`IsAdministrator && IsActive`). If `true`, immediately returns `true`.
5. **User-level evaluation**:
   - If the user type directly implements `HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool`, calls it.
   - Otherwise, if the user embeds `*users.Base`, delegates to `base.HasObjectPermission(r.Context(), obj, perms...)`.
   - If neither is implemented, triggers an assertion failure.

### Two-Tier Evaluation in `users.Base.HasObjectPermission`

Inside `users.Base.HasObjectPermission`, permission strings are evaluated with **AND semantics** (every string in `perms...` must be satisfied either directly on the user or through one of the user's groups):

1. **Preloaded Cache Tier**:
   `auth.AddUserMiddleware()` preloads `Permissions` and `Groups.Permissions` (`queries.NoJoins`) when loading the authenticated user from the session. `checkCachedPermissions` checks `u.Permissions.Cache()` first, then `group.Permissions.Cache()` for each preloaded group. If every requested permission is found in the in-memory caches, `HasObjectPermission` returns `true` without issuing any SQL queries.
2. **Single-Query Database Fallback**:
   If one or more requested permissions are not satisfied by the preloaded caches (for example, when querying a user manually without `.Preload(...)` or when permissions were assigned during the current request), `checkDatabasePermissions` builds one `Exists()` query combining an `expr.And(...)` over each remaining permission:
   - Each permission matches if an `EXISTS` subquery finds it either directly in `auth_user_permissions` or via a group in `auth_user_groups` + `auth_group_permissions`.

Do note that the database fallback relies on `u.Base.Fields(u)` having been included in your user model's `FieldDefs` (which binds the `backref` instance) and on the user model having a primary key field named `ID`.

---

## Implementing Object-Level Permissions

By default, `users.Base.HasObjectPermission` ignores `obj` and checks RBAC permission names only. However, because `PermissionsBackend` checks for `HasObjectPermission(context.Context, interface{}, ...string) bool` on your concrete user type before falling back to the embedded `users.Base`, custom user models can implement real per-object checks and fall back to `u.Base.HasObjectPermission`:

```go
type Article struct {
	ID       uint64
	AuthorID uint64
}

type CustomUser struct {
	models.Model `table:"custom_users"`
	users.Base
	ID uint64 `attrs:"primary;readonly"`
	// ...
}

func (u *CustomUser) FieldDefs(ctx context.Context) attrs.Definitions {
	return u.Model.Define(ctx, u,
		attrs.Unbound("ID"),
		u.Base.Fields(u),
	)
}

func (u *CustomUser) HasObjectPermission(ctx context.Context, obj interface{}, perms ...string) bool {
	if article, ok := obj.(*Article); ok && article.AuthorID == u.ID {
		for _, p := range perms {
			if p == "articles:edit" || p == "articles:delete" {
				return true
			}
		}
	}
	return u.Base.HasObjectPermission(ctx, obj, perms...)
}
```

---

## Creating and Assigning Permissions

### Seeding Permissions with `users.CreatePermissions`

Use `users.CreatePermissions` in startup hooks or migrations to idempotently create permission records inside a transaction:

```go
func CreatePermissions(ctx context.Context, permissions ...*Permission) (createdAny bool, err error)
```

It queries existing permissions by `Name__in`, bulk-inserts only missing entries, and returns `createdAny = true` if at least one new permission was created:

```go
created, err := users.CreatePermissions(ctx,
	&users.Permission{
		Name:        "admin:access_admin",
		Description: "Allows logging into the admin dashboard",
	},
	&users.Permission{
		Name:        "admin:view_app:blog",
		Description: "Allows viewing the blog app in admin",
	},
	&users.Permission{
		Name:        "admin:view_list",
		Description: "Allows viewing model lists in admin",
	},
)
if err != nil {
	return err
}
```

### Assigning via M2M QuerySets

You can assign permissions directly to users or groups using their `RelM2M` query sets:

```go
// Look up a permission
permRow, err := queries.GetQuerySet(&users.Permission{}).
	WithContext(ctx).
	Filter("Name", "admin:access_admin").
	First()
if err != nil {
	return err
}

// Assign directly to a user
if _, err := user.Permissions.Objects().WithContext(ctx).AddTarget(permRow.Object); err != nil {
	return err
}

// Or create/assign via a group
editorsGroup := models.Setup(ctx, &users.Group{Name: "Editors"})
editorsGroup, err = queries.GetQuerySet(&users.Group{}).WithContext(ctx).Create(editorsGroup)
if err != nil {
	return err
}

if _, err := editorsGroup.Permissions.Objects().WithContext(ctx).AddTarget(permRow.Object); err != nil {
	return err
}

if _, err := user.Groups.Objects().WithContext(ctx).AddTarget(editorsGroup); err != nil {
	return err
}
```

### Assigning via the Admin UI

When using [Admin](../admin/app.md), `auth` registers `User`, `Group`, and `Permission` under the `"Authentication and Authorization"` section:

- Create permissions under **Permissions** (matching the exact strings checked by your views or the [Admin Permission Strings](../admin/app.md#admin-permission-strings)).
- Bundle permissions into roles under **Groups**.
- Assign `Groups` and individual `Permissions` on each user's edit page (or toggle `IsAdministrator` to bypass RBAC checks entirely).
