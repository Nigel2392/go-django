package auth_test

import (
	"context"
	stderrors "errors"
	"flag"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "unsafe"
)

func TestMain(m *testing.M) {
	testing.Init()

	flag.Parse()

	var which, db = testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE:       db,
		django.APPVAR_DISABLE_NOSURF: true,
		django.APPVAR_ALLOWED_HOSTS:  []string{"*"},
	}

	var appLogger = &logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	}
	logger.Setup(appLogger)

	if !testing.Verbose() {
		appLogger.SetLevel(logger.WRN)
	}

	var app = django.App(
		django.Configure(settings),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
		),
		django.Flag(
			django.FlagSkipCmds,
			django.FlagSkipChecks,
			django.FlagSkipDepsCheck,
		),
		django.AppLogger(appLogger),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}

	tpl.Add(tpl.Config{
		AppName: "auth",
		FS: fstest.MapFS{
			"auth/login.tmpl": {
				Data: []byte(`{{ define "login" }}<form method="post" action="{{ .Get "FormURL" }}"><input type="hidden" name="next" value="{{ .Get "NextURL" }}">{{ $f := .Get "Form" }}{{ range $f.ErrorList }}<div class="form-error">{{ . }}</div>{{ end }}{{ $f.AsP }}</form>{{ end }}`),
			},
			"auth/register.tmpl": {
				Data: []byte(`{{ define "register" }}<form method="post" action="{{ .Get "FormURL" }}"><input type="hidden" name="next" value="{{ .Get "NextURL" }}">{{ $f := .Get "Form" }}{{ range $f.ErrorList }}<div class="form-error">{{ . }}</div>{{ end }}{{ $f.AsP }}</form>{{ end }}`),
			},
		},
		Matches: filesystem.MatchPrefix("auth/"),
	})

	logger.Debugf("Using %s database for queries tests", which)

	var tables = quest.Table[*testing.T](nil,
		&auth.User{},
		&users.Group{},
		&users.Permission{},
		&users.UserGroup{},
		&users.GroupPermission{},
		&users.UserPermission{},
	)
	tables.IfNotExists = true
	tables.Create()

	attrs.ResetDefinitions.Send(context.Background(), nil)

	exitCode := m.Run()
	if exitCode != 0 {
		// If the test run failed, we can log an error or perform cleanup if necessary.
		// For now, we just exit with the code returned by m.Run().
	}

	tables.Drop()
	os.Exit(exitCode)
}

func TestUserPassword(t *testing.T) {
	var user = models.Setup(t.Context(), &auth.User{
		Email:     drivers.MustParseEmail("test1@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser1",
		FirstName: "Test",
		LastName:  "User",
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	})

	if err := user.Password.Check("testpassword"); err != nil {
		t.Fatalf("Password check failed: %v", err)
	}

	user.Password = auth.NewPassword("testpassword")
	if err := user.Save(context.Background()); err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	var userRow, err = auth.GetUserQuerySet().WithContext(context.Background()).Filter("ID", user.ID).First()
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
		return
	}

	if userRow.Object.Password.IsZero() {
		t.Fatalf("User password should not be zero: %#v", userRow.Object.Password)
		return
	}

	if err := userRow.Object.Password.Check("testpassword"); err != nil {
		t.Fatalf("Password check failed after save: %v", err)
		return
	}
}

func TestUserAddGroups(t *testing.T) {
	var ctx = context.Background()
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to start transaction: %v", err)
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}
	}()

	var u = &auth.User{
		Email:     drivers.MustParseEmail("test2@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser2",
		FirstName: "Test",
		LastName:  "User",
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	}
	user, err := auth.GetUserQuerySet().WithContext(ctx).Create(u)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
		return
	}

	var userList = make([]*auth.User, 0, 5)
	for i := 0; i < 5; i++ {
		var user = &auth.User{
			Email:     drivers.MustParseEmail("test_" + drivers.Uint(i+1).String() + "@example.com"),
			Password:  auth.NewPassword("testpassword" + drivers.Uint(i+1).String()),
			Username:  "testuser__" + drivers.Uint(i+1).String(),
			FirstName: "Test" + drivers.Uint(i+1).String(),
			LastName:  "User" + drivers.Uint(i+1).String(),
			Base: users.Base{
				IsAdministrator: false,
				IsActive:        true,
			},
		}
		user, err := auth.GetUserQuerySet().WithContext(ctx).Create(user)
		if err != nil {
			t.Fatalf("Failed to create user %d: %v", i+1, err)
			return
		}
		userList = append(userList, user)
	}

	t.Logf("User %s created with ID %d", user.Username, user.ID)

	_, err = user.Groups.Objects().WithContext(ctx).AddTargets(
		&users.Group{
			Name: "Administrators",
		},
		&users.Group{
			Name: "Moderators",
		},
		&users.Group{
			Name: "Viewers",
		},
	)
	if err != nil {
		t.Fatalf("Failed to add groups to user: %v", err)
	}

	_, err = user.Permissions.Objects().WithContext(ctx).AddTargets(
		&users.Permission{
			Name:        "Can view users",
			Description: "Allows viewing of users",
		},
		&users.Permission{
			Name:        "Can edit users",
			Description: "Allows editing of users",
		},
		&users.Permission{
			Name:        "Can delete users",
			Description: "Allows deletion of users",
		},
	)
	if err != nil {
		t.Fatalf("Failed to add permissions to user: %v", err)
	}

	// add 3 perms to admin group
	_, err = user.Groups.AsList()[0].Object.Permissions.Objects().WithContext(ctx).AddTargets(
		&users.Permission{
			Name:        "Can view users",
			Description: "Allows viewing of users",
		},
		&users.Permission{
			Name:        "Can edit users",
			Description: "Allows editing of users",
		},
		&users.Permission{
			Name:        "Can delete users",
			Description: "Allows deletion of users",
		},
	)
	if err != nil {
		t.Fatalf("Failed to add permissions to user: %v", err)
	}

	t.Log("\n\n\n")

	t.Run("GroupsPreloaded", func(t *testing.T) {
		userRows, err := auth.GetUserQuerySet().
			WithContext(ctx).
			Select("*").
			Preload("Permissions", "Groups.Permissions.GroupPermissions").
			Filter("ID__in", append(userList, user)).
			OrderBy("ID", "Groups.Name", "Permissions.Name").
			All()
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
			return
		}

		for i, userRow := range userRows {
			t.Logf("User %s with ID %d has groups:", userRow.Object.Username, userRow.Object.ID)
			for _, group := range userRow.Object.Groups.AsList() {
				t.Logf(" - Group: %s", group.Object.Name)
				for _, perm := range group.Object.Permissions.AsList() {
					t.Logf("   - Permission: %s", perm.Object.Name)
					var groupPerms = perm.Object.GroupPermissions

					for _, groupPerm := range groupPerms.AsList() {
						t.Logf("     - Group Permission: %s", groupPerm.Object.Name)
					}
				}
			}

			t.Logf("User %s has permissions:", userRow.Object.Username)
			for _, perm := range userRow.Object.Permissions.AsList() {
				t.Logf(" - Permission: %s", perm.Object.Name)
			}

			if i == 0 {
				if l := len(userRow.Object.Groups.AsList()); l < 3 {
					t.Fatalf("[%d] Expected at least 3 Groups, got %d", i, l)
				}
			}

			for i, group := range userRow.Object.Groups.AsList() {
				switch i {
				case 0:
					if group.Object.Name != "Administrators" {
						t.Fatalf("Expected group name 'Administrators', got '%s'", group.Object.Name)
					}
				case 1:
					if group.Object.Name != "Moderators" {
						t.Fatalf("Expected group name 'Moderators', got '%s'", group.Object.Name)
					}
				case 2:
					if group.Object.Name != "Viewers" {
						t.Fatalf("Expected group name 'Viewers', got '%s'", group.Object.Name)
					}
				default:
					t.Fatalf("Unexpected group index %d", i)
				}
			}

			if i == 0 {
				if l := len(userRow.Object.Permissions.AsList()); l < 3 {
					t.Fatalf("[%d] Expected at least 3 Permissions, got %d", i, l)
				}
			}

			for i, perm := range userRow.Object.Permissions.AsList() {
				switch i {
				case 0:
					if perm.Object.Name != "Can view users" {
						t.Fatalf("Expected permission name 'Can view users', got '%s'", perm.Object.Name)
					}
				case 1:
					if perm.Object.Name != "Can edit users" {
						t.Fatalf("Expected permission name 'Can edit users', got '%s'", perm.Object.Name)
					}
				case 2:
					if perm.Object.Name != "Can delete users" {
						t.Fatalf("Expected permission name 'Can delete users', got '%s'", perm.Object.Name)
					}
				default:
					t.Fatalf("Unexpected permission index %d", i)
				}
			}
		}

	})

	t.Log("\n\n\n")

	t.Run("GroupsFromQuerySet", func(t *testing.T) {
		userRow, err := auth.GetUserQuerySet().
			WithContext(ctx).
			Select("*", "Groups.*").
			Filter("ID", user.ID).
			OrderBy("ID", "Groups.Name").
			First()
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
			return
		}

		for i, group := range userRow.Object.Groups.AsList() {
			switch i {
			case 0:
				if group.Object.Name != "Administrators" {
					t.Fatalf("Expected group name 'Administrators', got '%s'", group.Object.Name)
				}
			case 1:
				if group.Object.Name != "Moderators" {
					t.Fatalf("Expected group name 'Moderators', got '%s'", group.Object.Name)
				}
			case 2:
				if group.Object.Name != "Viewers" {
					t.Fatalf("Expected group name 'Viewers', got '%s'", group.Object.Name)
				}
			default:
				t.Fatalf("Unexpected group index %d", i)
			}
		}
	})

	t.Run("GroupsFromAttribute", func(t *testing.T) {
		userRow, err := auth.GetUserQuerySet().WithContext(ctx).Filter("ID", user.ID).First()
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
			return
		}

		groups, err := userRow.Object.Groups.Objects().WithContext(ctx).OrderBy("Name").All()
		if err != nil {
			t.Fatalf("Failed to retrieve user groups: %v", err)
			return
		}

		if len(groups) != 3 {
			t.Fatalf("Expected 3 groups, got %d", len(groups))
			return
		}

		for i, group := range groups {
			switch i {
			case 0:
				if group.Object.Name != "Administrators" {
					t.Fatalf("Expected group name 'Administrators', got '%s'", group.Object.Name)
				}
			case 1:
				if group.Object.Name != "Moderators" {
					t.Fatalf("Expected group name 'Moderators', got '%s'", group.Object.Name)
				}
			case 2:
				if group.Object.Name != "Viewers" {
					t.Fatalf("Expected group name 'Viewers', got '%s'", group.Object.Name)
				}
			default:
				t.Fatalf("Unexpected group index %d", i)
			}
		}
	})
}

func TestUserAddPermissions(t *testing.T) {
	var ctx = context.Background()
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to start transaction: %v", err)
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}
	}()

	var u = &auth.User{
		Email:     drivers.MustParseEmail("test3@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser3",
		FirstName: "Test",
		LastName:  "User",
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	}
	user, err := auth.GetUserQuerySet().WithContext(ctx).Create(u)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
		return
	}

	_, err = user.Permissions.Objects().WithContext(ctx).AddTargets(
		&users.Permission{
			Name:        "Can view users",
			Description: "Allows viewing of users",
		},
		&users.Permission{
			Name:        "Can edit users",
			Description: "Allows editing of users",
		},
		&users.Permission{
			Name:        "Can delete users",
			Description: "Allows deletion of users",
		},
	)
	if err != nil {
		t.Fatalf("Failed to add permissions to user: %v", err)
	}

	userRow, err := auth.GetUserQuerySet().WithContext(ctx).Filter("ID", user.ID).First()
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	permissions, err := userRow.Object.Permissions.Objects().WithContext(ctx).OrderBy("Name").All()
	if err != nil {
		t.Fatalf("Failed to retrieve user permissions: %v", err)
		return
	}

	if len(permissions) != 3 {
		t.Fatalf("Expected 3 permissions, got %d", len(permissions))
		return
	}

	for i, perm := range permissions {
		switch i {
		case 0:
			if perm.Object.Name != "Can delete users" {
				t.Fatalf("Expected permission name 'Can delete users', got '%s'", perm.Object.Name)
			}
		case 1:
			if perm.Object.Name != "Can edit users" {
				t.Fatalf("Expected permission name 'Can edit users', got '%s'", perm.Object.Name)
			}
		case 2:
			if perm.Object.Name != "Can view users" {
				t.Fatalf("Expected permission name 'Can view users', got '%s'", perm.Object.Name)
			}
		default:
			t.Fatalf("Unexpected permission index %d", i)
		}
	}
}

//go:linkname cacheFlags github.com/Nigel2392/go-django/contrib/auth/users.cacheFlags
func cacheFlags(ctx context.Context, disableCache bool, disableDB bool) context.Context

func TestUserHasPermissions(t *testing.T) {
	var ctx = context.Background()
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to start transaction: %v", err)
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}
	}()

	var u = &auth.User{
		Email:     drivers.MustParseEmail("test4@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser4",
		FirstName: "Test",
		LastName:  "User",
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	}
	user, err := auth.GetUserQuerySet().
		WithContext(ctx).
		Create(u)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
		return
	}

	_, err = user.Groups.Objects().
		WithContext(ctx).
		AddTarget(&users.Group{
			Name: "Administrators",
		})
	if err != nil {
		t.Fatalf("Failed to add group to user: %v", err)
	}

	_, err = user.Permissions.Objects().
		WithContext(ctx).
		AddTargets(
			&users.Permission{
				Name:        "Can view users",
				Description: "Allows viewing of users",
			},
			&users.Permission{
				Name:        "Can edit users",
				Description: "Allows editing of users",
			},
		)
	if err != nil {
		t.Fatalf("Failed to add permissions to user: %v", err)
	}

	_, err = user.Groups.AsList()[0].Object.Permissions.Objects().
		WithContext(ctx).
		AddTargets(&users.Permission{
			Name:        "Can delete users",
			Description: "Allows deletion of users",
		})
	if err != nil {
		t.Fatalf("Failed to add permissions to group: %v", err)
	}

	t.Run("WithPreload", func(t *testing.T) {
		userRow, err := auth.GetUserQuerySet().
			WithContext(ctx).
			Preload("Permissions", "Groups.Permissions").
			Filter("ID", user.ID).
			First()
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
		}

		u = userRow.Object

		t.Logf("Checking permissions for user: %s", u.Username)

		var contextWithFlags = cacheFlags(ctx, false, true)
		if !u.HasObjectPermission(contextWithFlags, nil, "Can view users", "Can edit users") {
			t.Errorf("User should have 'Can view users' and 'Can edit users' permissions, but does not")
		}

		if !u.HasObjectPermission(contextWithFlags, nil, "Can delete users") {
			t.Errorf("User should have 'Can delete users' permission, but does not")
		}

		if u.HasObjectPermission(contextWithFlags, nil, "Can create users") {
			t.Errorf("User should not have 'Can create users' permission, but does")
		}
	})

	t.Run("WithoutPreload", func(t *testing.T) {
		userRow, err := auth.GetUserQuerySet().
			WithContext(ctx).
			Filter("ID", user.ID).
			First()
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
		}

		u = userRow.Object

		t.Logf("Checking permissions for user: %s", u.Username)

		if !u.HasObjectPermission(ctx, nil, "Can view users", "Can edit users") {
			t.Errorf("User should have 'Can view users' and 'Can edit users' permissions, but does not")
		}

		if !u.HasObjectPermission(ctx, nil, "Can delete users") {
			t.Errorf("User should have 'Can delete users' permission, but does not")
		}

		if u.HasObjectPermission(ctx, nil, "Can create users") {
			t.Errorf("User should not have 'Can create users' permission, but does")
		}
	})
}

func TestUser_StringAndEdgeCases(t *testing.T) {
	ctx := t.Context()

	t.Run("String", func(t *testing.T) {
		noName := &auth.User{ID: 42}
		if got := noName.String(); got != "User<42>" {
			t.Errorf("User.String() with empty username = %q, want %q", got, "User<42>")
		}

		withName := &auth.User{ID: 42, Username: "alice"}
		if got := withName.String(); got != "alice" {
			t.Errorf("User.String() with username = %q, want %q", got, "alice")
		}
	})

	t.Run("UniqueTogetherAndIndexes", func(t *testing.T) {
		u := &auth.User{}
		ut := u.UniqueTogether()
		if len(ut) != 2 || len(ut[0]) != 1 || ut[0][0] != "Email" || len(ut[1]) != 1 || ut[1][0] != "Username" {
			t.Errorf("unexpected UniqueTogether: %#v", ut)
		}

		idxs := u.DatabaseIndexes(u)
		if len(idxs) != 2 {
			t.Fatalf("expected 2 indexes, got %d", len(idxs))
		}
		if idxs[0].Identifier != "auth_users_email_idx" || !idxs[0].Unique || idxs[0].Fields[0] != "Email" {
			t.Errorf("unexpected email index: %#v", idxs[0])
		}
		if idxs[1].Identifier != "auth_users_username_idx" || !idxs[1].Unique || idxs[1].Fields[0] != "Username" {
			t.Errorf("unexpected username index: %#v", idxs[1])
		}
	})

	t.Run("BeforeCreateAndBeforeSave", func(t *testing.T) {
		u := &auth.User{
			Email:    drivers.MustParseEmail("edge_beforesave@example.com"),
			Username: "edge_beforesave",
		}
		if err := u.BeforeCreate(ctx); err != nil {
			t.Fatalf("BeforeCreate error: %v", err)
		}
		if time.Time(u.CreatedAt).IsZero() {
			t.Errorf("expected CreatedAt to be set by BeforeCreate")
		}

		if err := u.BeforeSave(ctx); !stderrors.Is(err, errors.ValueError) {
			t.Errorf("expected errors.ValueError when password is nil, got %v", err)
		}

		u.Password = auth.NewPassword("")
		if err := u.BeforeSave(ctx); !stderrors.Is(err, errors.ValueError) {
			t.Errorf("expected errors.ValueError when password is zero, got %v", err)
		}

		u.SetPassword("ValidPass1!")
		if err := u.BeforeSave(ctx); err != nil {
			t.Fatalf("BeforeSave with valid password error: %v", err)
		}
		if time.Time(u.UpdatedAt).IsZero() {
			t.Errorf("expected UpdatedAt to be set by BeforeSave")
		}
	})

	t.Run("CloneQuerySetAndUniqueViolation", func(t *testing.T) {
		qs := auth.GetUserQuerySet()
		cloned := qs.WithContext(ctx).Filter("Username", "nonexistent_clone_check")
		if cloned == qs {
			t.Errorf("expected cloned QuerySet to be a distinct instance")
		}

		ctxTx, tx, err := queries.StartTransaction(ctx)
		if err != nil {
			t.Fatalf("StartTransaction: %v", err)
		}
		defer tx.Rollback(ctxTx)

		_, err = auth.GetUserQuerySet().WithContext(ctxTx).Create(&auth.User{
			Email:    drivers.MustParseEmail("dup_user@example.com"),
			Username: "dup_user",
			Password: auth.NewPassword("Secret123!"),
			Base:     users.Base{IsActive: true},
		})
		if err != nil {
			t.Fatalf("create first user: %v", err)
		}

		_, err = auth.GetUserQuerySet().WithContext(ctxTx).Create(&auth.User{
			Email:    drivers.MustParseEmail("dup_user@example.com"),
			Username: "dup_user_other",
			Password: auth.NewPassword("Secret123!"),
			Base:     users.Base{IsActive: true},
		})
		if err == nil {
			t.Errorf("expected duplicate email creation to fail with unique index error")
		}

		_, err = auth.GetUserQuerySet().WithContext(ctxTx).Create(&auth.User{
			Email:    drivers.MustParseEmail("dup_user_other@example.com"),
			Username: "dup_user",
			Password: auth.NewPassword("Secret123!"),
			Base:     users.Base{IsActive: true},
		})
		if err == nil {
			t.Errorf("expected duplicate username creation to fail with unique index error")
		}
	})
}
