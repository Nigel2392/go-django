package auth_test

import (
	"context"
	"flag"
	"os"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"

	_ "unsafe"
)

func TestMain(m *testing.M) {
	testing.Init()

	flag.Parse()

	var which, db = testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(django.Configure(settings))

	logger.Debugf("Using %s database for queries tests", which)

	if !testing.Verbose() {
		logger.SetLevel(logger.WRN)
	}

	var tables = quest.Table[*testing.T](nil,
		&auth.User{},
		&users.Group{},
		&users.Permission{},
		&users.UserGroup{},
		&users.GroupPermission{},
		&users.UserPermission{},
	)
	tables.Create()
	defer tables.Drop()

	exitCode := m.Run()
	if exitCode != 0 {
		// If the test run failed, we can log an error or perform cleanup if necessary.
		// For now, we just exit with the code returned by m.Run().
	}

	os.Exit(exitCode)
}

func TestUserPassword(t *testing.T) {
	var user = models.Setup(&auth.User{
		Email:     drivers.MustParseEmail("test@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser",
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
		Email:     drivers.MustParseEmail("test@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser",
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
			Email:     drivers.MustParseEmail("test" + drivers.Uint(i+1).String() + "@example.com"),
			Password:  auth.NewPassword("testpassword" + drivers.Uint(i+1).String()),
			Username:  "testuser" + drivers.Uint(i+1).String(),
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
			Preload("Permissions", "Groups", "Groups.Permissions", "Groups.Permissions.GroupPermissions").
			Filter("ID__in", append(userList, user)).
			OrderBy("ID", "Groups.Name", "Permissions.Name").
			All()
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
			return
		}

		for _, userRow := range userRows {
			t.Logf("User %s with ID %d has groups:", userRow.Object.Username, userRow.Object.ID)
			for _, group := range userRow.Object.Groups.AsList() {
				t.Logf(" - Group: %s", group.Object.Name)
				for _, perm := range group.Object.Permissions.AsList() {
					t.Logf("   - Permission: %s", perm.Object.Name)
					var groupPermsObj, ok = perm.Object.DataStore().GetValue("GroupPermissions")
					if ok {
						var groupPerms, ok = groupPermsObj.(*queries.RelM2M[attrs.Definer, attrs.Definer])
						if !ok {
							t.Fatalf("Expected GroupPermissions to be of type *queries.RelM2M, got %T", groupPermsObj)
						}

						for _, groupPerm := range groupPerms.AsList() {
							t.Logf("     - Group Permission: %s", groupPerm.Object.(*users.Group).Name)
						}
					}
				}
			}
			t.Logf("User %s has permissions:", userRow.Object.Username)
			for _, perm := range userRow.Object.Permissions.AsList() {
				t.Logf(" - Permission: %s", perm.Object.Name)
			}
		}

		//for i, group := range userRow.Object.Groups.AsList() {
		//	switch i {
		//	case 0:
		//		if group.Object.Name != "Administrators" {
		//			t.Fatalf("Expected group name 'Administrators', got '%s'", group.Object.Name)
		//		}
		//	case 1:
		//		if group.Object.Name != "Moderators" {
		//			t.Fatalf("Expected group name 'Moderators', got '%s'", group.Object.Name)
		//		}
		//	case 2:
		//		if group.Object.Name != "Viewers" {
		//			t.Fatalf("Expected group name 'Viewers', got '%s'", group.Object.Name)
		//		}
		//	default:
		//		t.Fatalf("Unexpected group index %d", i)
		//	}
		//}
		//
		//for i, perm := range userRow.Object.Permissions.AsList() {
		//	switch i {
		//	case 0:
		//		if perm.Object.Name != "Can view users" {
		//			t.Fatalf("Expected permission name 'Can view users', got '%s'", perm.Object.Name)
		//		}
		//	case 1:
		//		if perm.Object.Name != "Can edit users" {
		//			t.Fatalf("Expected permission name 'Can edit users', got '%s'", perm.Object.Name)
		//		}
		//	case 2:
		//		if perm.Object.Name != "Can delete users" {
		//			t.Fatalf("Expected permission name 'Can delete users', got '%s'", perm.Object.Name)
		//		}
		//	default:
		//		t.Fatalf("Unexpected permission index %d", i)
		//	}
		//}

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
		Email:     drivers.MustParseEmail("test@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser",
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

//go:linkname cacheFlags github.com/Nigel2392/go-django/src/contrib/auth/users.cacheFlags
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
		Email:     drivers.MustParseEmail("test@example.com"),
		Password:  auth.NewPassword("testpassword"),
		Username:  "testuser",
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
			Preload("Permissions", "Groups", "Groups.Permissions").
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
