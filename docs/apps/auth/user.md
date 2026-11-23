# User Model, Passwords, and Custom Users

This page covers the built-in `auth.User` model, bcrypt-backed password hashing (`securedb.StringHash`), and the `contrib/auth/users` abstractions for custom user models, deferred user relations, groups, and permissions.

For authentication setup and RBAC evaluation, see:

- [Authentication App (`auth`)](./app.md)
- [Permissions Backend and RBAC](./permissions.md)

---

## Built-in `auth.User`

The default user model is `auth.User`, mapped to the `auth_users` table. It embeds `models.Model` and `users.Base`:

```go
type User struct {
	models.Model `table:"auth_users" json:"-"`
	users.Base

	ID        uint64           `json:"id" attrs:"primary;readonly"`
	Email     *drivers.Email   `json:"email"`
	Username  string           `json:"username"`
	Password  *Password        `json:"-"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	CreatedAt drivers.DateTime `json:"created_at" attrs:"readonly"`
	UpdatedAt drivers.DateTime `json:"updated_at" attrs:"readonly"`
}
```

### Fields and Constraints

- `ID`: Primary key (`id`), read-only `uint64`.
- `Email`: `*drivers.Email` (`email`, `MaxLength: 255`, `MinLength: 3`), uses `fields.EmailField` in forms.
- `Username`: `string` (`username`, `MaxLength: 16`, `MinLength: 3`).
- `Password`: `*auth.Password` (`password`, `MaxLength: 255`), stored as a bcrypt hash.
- `FirstName`, `LastName`: `string` (`first_name`, `last_name`, `MaxLength: 75`).
- `CreatedAt`, `UpdatedAt`: `drivers.DateTime` (`created_at`, `updated_at`), automatically populated by `BeforeCreate` and `BeforeSave`.
- Embedded `users.Base`: Adds `IsAdministrator`, `IsActive`, `IsLoggedIn` (runtime-only), `LastLogin`, `Groups`, and `Permissions`.

`auth.User` defines unique constraints and unique btree indexes on both `Email` (`auth_users_email_idx`) and `Username` (`auth_users_username_idx`):

- `BeforeCreate`: Sets `u.CreatedAt = drivers.CurrentDateTime()`.
- `BeforeSave`: Rejects zero/empty `u.Password` with `errors.ValueError` and sets `u.UpdatedAt = drivers.CurrentDateTime()`.
- `SetPassword(password string) *User`: Assigns `u.Password = NewPassword(password)` and returns `u` for chaining.
- `String() string`: Returns `u.Username`, or `fmt.Sprintf("User<%d>", u.ID)` when `Username` is empty.

### Querying Users with `UserQuerySet`

Use `auth.GetUserQuerySet()` (or `queries.GetQuerySet(&auth.User{})`) to query and manage users:

```go
user := models.Setup(ctx, &auth.User{
	Email:     drivers.MustParseEmail("alice@example.com"),
	Username:  "alice",
	FirstName: "Alice",
	LastName:  "Example",
	Base: users.Base{
		IsActive:        true,
		IsAdministrator: false,
	},
}).SetPassword("SecretPass1!")

created, err := auth.GetUserQuerySet().WithContext(ctx).Create(user)
```

---

## Password Hashing and `securedb`

`auth.Password` is a type alias for `securedb.StringHash[BcryptHasher]` from `github.com/Nigel2392/go-django/queries/src/drivers/securedb`:

```go
type BcryptHasher struct{}

func (h BcryptHasher) HashValue(b string) (string, error)
func (h BcryptHasher) CheckValueHash(hashedPassword, value string) error

type Password = securedb.StringHash[BcryptHasher]

func NewPassword(raw string) *Password
func NewHashedPassword(hash string) *Password
func CheckPassword(u *User, password string) error
```

Key behaviors of `Password`:

- **Lazy hashing on save**: `NewPassword(raw)` stores `Raw` without immediately running bcrypt. When `Value()` is called during `Create`/`Save`/`Update`, `StringHash` hashes `Raw` via `BcryptHasher` (`bcrypt.DefaultCost`) and caches the result in `Hash`.
- **Scanning from the database**: `Scan(...)` reads `string`, `[]byte`, or `StringHash[BcryptHasher]` directly into `Hash` without re-hashing.
- **Verification**: `CheckPassword(u, raw)` calls `u.Password.Check(raw)`. On mismatch, `BcryptHasher.CheckValueHash` returns `autherrors.ErrPwdHashMismatch` (`autherrors.ErrPasswordInvalid`).
- **Safe formatting**: `String()` returns `"<nil>"` when zero, the stored `Hash` if already hashed, or `"***********"` when holding only an unhashed `Raw` password.

### Custom Password Hashers

You can implement your own password type for custom user models by satisfying `securedb.HasherType`:

```go
type HasherType interface {
	HashValue(b string) (string, error)
	CheckValueHash(hashedPassword, value string) error
}
```

For example:

```go
type Argon2Hasher struct{}

func (h Argon2Hasher) HashValue(raw string) (string, error) {
	// Return encoded Argon2id hash string.
	return hashWithArgon2(raw)
}

func (h Argon2Hasher) CheckValueHash(hashedPassword, raw string) error {
	if !verifyArgon2(hashedPassword, raw) {
		return autherrors.ErrPwdHashMismatch
	}
	return nil
}

type CustomPassword = securedb.StringHash[Argon2Hasher]
```

---

## Custom Users and `contrib/auth/users`

The `github.com/Nigel2392/go-django/contrib/auth/users` package decouples user models, groups, permissions, and foreign-key references from the concrete `auth.User` struct. This enables alternative auth backends such as [OAuth2](../oauth2.md) or application-specific user tables.

### The `users.User` Interface and Lazy Resolution

```go
type User interface {
	attrs.Definer
	authentication.User
}

const MODEL_KEY = "users.User"

func GetUserModel(typeNames ...string) *contenttypes.ContentTypeDefinition
func GetUserModelString(typeNames ...string) string
```

Any model implementing `attrs.Definer`, `IsAuthenticated() bool`, and `IsAdmin() bool` (or exposing an embedded `*users.Base`) satisfies the lazy registry keyed by `users.MODEL_KEY` (`"users.User"`).

### Referencing the Active User Model (`attrs.RelatedDeferred`)

When another model (such as a blog post or audit log entry) has a foreign key to "the project's user model", use `attrs.RelatedDeferred` with `users.MODEL_KEY` rather than hardcoding `&auth.User{}`:

```go
package blog

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type Post struct {
	models.Model `table:"blog_posts"`
	ID           int64      `attrs:"primary;readonly"`
	Title        string
	Author       users.User
}

func (p *Post) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID"),
		attrs.Unbound("Title"),
		attrs.NewField(p, "Author", &attrs.Config{
			Label: trans.S("Author"),
			Null:  true,
			RelForeignKey: attrs.RelatedDeferred(
				attrs.RelManyToOne,
				users.MODEL_KEY,
				"", nil,
			),
			FormWidget: func(fc attrs.Config) widgets.Widget {
				return chooser.NewChooserWidget(
					fc.RelForeignKey.Model(), fc.WidgetAttrs,
				)
			},
		}),
	)
}
```

At startup, Go-Django resolves `users.MODEL_KEY` via [Content Types](../../contenttypes.md) to whichever `users.User` implementation is registered in your application.

### Embedding `users.Base` in a Custom User Model

If you want built-in `IsAdministrator`, `IsActive`, `LastLogin`, `Groups`, `Permissions`, and RBAC support on a custom user struct, embed `users.Base` and **include `u.Base.Fields(u)`** inside `FieldDefs`:

```go
type Base struct {
	IsAdministrator bool      `json:"is_administrator" attrs:"blank"`
	IsActive        bool      `json:"is_active" attrs:"blank;default=true"`
	IsLoggedIn      bool      `json:"is_logged_in"`
	LastLogin       time.Time `json:"last_login"`

	Groups      *queries.RelM2M[*Group, *UserGroup]           `json:"-"`
	Permissions *queries.RelM2M[*Permission, *UserPermission] `json:"-"`
	// ...
}

func (u *Base) Fields(user attrs.Definer) []any
func (u *Base) IsAuthenticated() bool // returns u.IsLoggedIn && u.IsActive
func (u *Base) IsAdmin() bool         // returns u.IsAdministrator && u.IsActive
func (u *Base) LastLoggedIn() time.Time
```

Calling `u.Base.Fields(u)` in `FieldDefs` is mandatory when using `users.Base`:

1. It binds an internal back-reference (`backref`) to your concrete user struct so database permission fallbacks can query the user's primary key. Calling database permission checks without `u.Base.Fields(u)` panics (`"Base backref is not set"`).
2. It registers the `IsAdministrator`, `IsActive`, `LastLogin`, `Groups`, and `Permissions` field definitions on your model.
3. Note that both `IsAuthenticated()` and `IsAdmin()` require `IsActive == true`. An inactive user—even with `IsAdministrator = true`—returns `false` for both methods.

```go
type Account struct {
	models.Model `table:"accounts"`
	users.Base
	ID          uint64 `attrs:"primary;readonly"`
	Subject     string `attrs:"max_length=255"`
	DisplayName string
}

func (a *Account) FieldDefs(ctx context.Context) attrs.Definitions {
	return a.Model.Define(ctx, a,
		attrs.Unbound("ID"),
		attrs.Unbound("Subject"),
		attrs.Unbound("DisplayName"),
		a.Base.Fields(a),
	)
}
```

---

## Groups, Permissions, and Through Models

The `contrib/auth/users` package also defines the RBAC models shared across authentication backends:

| Model | Table Name | Key Fields & Relations |
| :--- | :--- | :--- |
| `users.Group` | `auth_groups` | `ID`, `Name`, `Description`, `Permissions` (`*queries.RelM2M[*Permission, *GroupPermission]`), `UserGroups` (`*queries.RelM2M[User, *UserGroup]`) |
| `users.Permission` | `auth_permissions` | `ID`, `Name`, `Description`, `GroupPermissions` (`*queries.RelM2M[*Group, *GroupPermission]`), `UserPermissions` (`*queries.RelM2M[User, *UserPermission]`) |
| `users.UserGroup` | `auth_user_groups` | `UserID`, `GroupID` (`UniqueTogether: [["UserID", "GroupID"]]`) |
| `users.UserPermission` | `auth_user_permissions` | `UserID`, `PermissionID` (`UniqueTogether: [["UserID", "PermissionID"]]`) |
| `users.GroupPermission` | `auth_group_permissions` | `GroupID`, `PermissionID` (`UniqueTogether: [["GroupID", "PermissionID"]]`) |

All three through models map to the user model via `users.MODEL_KEY`, so custom user models that embed `users.Base` (with a `uint64`/`drivers.Uint` primary key named `ID`) work directly with `Group` and `Permission` M2M queries. For details on creating, assigning, and evaluating permissions, see [Permissions Backend and RBAC](./permissions.md).
