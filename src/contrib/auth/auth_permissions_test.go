package auth_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/auth"
	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	auth_permissions "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions"
	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	django_models "github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux/middleware/authentication"

	_ "unsafe"

	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-models/auth-models-sqlite"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-sqlite"
	_ "github.com/mattn/go-sqlite3"
)

const context_user_key = "mux.middleware.authentication.User"

var (
	db            *sql.DB
	q             models.DBQuerier
	pq            auth_permissions.DBQuerier
	b             django_models.Backend[models.Querier]
	pb            django_models.Backend[permissions_models.Querier]
	users         []models.User
	permissionMap = map[uint64][]string{
		1:  {"auth.add_user", "auth.change_user", "auth.delete_user"},
		2:  {"auth.add_group", "auth.change_group", "auth.delete_group"},
		3:  {"auth.add_permission", "auth.change_permission", "auth.delete_permission"},
		4:  {"auth.add_user", "auth.change_user", "auth.delete_user"},
		5:  {"auth.add_group", "auth.change_group", "auth.delete_group"},
		6:  {"auth.add_permission", "auth.change_permission", "auth.delete_permission"},
		7:  {"auth.add_user", "auth.change_user", "auth.delete_user"},
		8:  {"auth.add_group", "auth.change_group", "auth.delete_group"},
		9:  {"auth.add_permission", "auth.change_permission", "auth.delete_permission"},
		10: {"auth.add_user", "auth.change_user", "auth.delete_user"},
	}
	permissionNamesMap = map[string]uint64{}
	permissionIdsMap   = make(map[uint64][]uint64)
)

func init() {
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	if q, err = models.NewQueries(db); err != nil {
		panic(err)
	}

	if pq, err = auth_permissions.NewQueries(db); err != nil {
		panic(err)
	}

	if b, err = models.BackendForDB(db.Driver()); err != nil {
		panic(err)
	}

	if pb, err = permissions_models.BackendForDB(db.Driver()); err != nil {
		panic(err)
	}

	if err := b.CreateTable(db); err != nil {
		panic(err)
	}

	if err := pb.CreateTable(db); err != nil {
		panic(err)
	}

	permissions.Tester = auth_permissions.NewPermissionsBackend(
		pq,
	)

	var usernameFmt = "user%d"
	var emailFmt = "user%d@example.com"
	var passwordFmt = "password%d"
	users = make([]models.User, 0, len(permissionMap))
	for i := 0; i < len(permissionMap); i++ {
		var email, _ = mail.ParseAddress(
			fmt.Sprintf(emailFmt, i),
		)
		var u = models.User{
			Username:   fmt.Sprintf(usernameFmt, i),
			Email:      (*models.Email)(email),
			IsLoggedIn: true,
			IsActive:   true,
		}
		if err = auth.SetPassword(
			&u, fmt.Sprintf(passwordFmt, i),
		); err != nil {
			panic(err)
		}

		var result int64
		var p = permissionMap[uint64(i+1)]
		for _, perm := range p {
			if _, ok := permissionNamesMap[perm]; ok {
				permissionIdsMap[uint64(i+1)] = append(
					permissionIdsMap[uint64(i+1)], permissionNamesMap[perm],
				)
				continue
			}

			var permObj = permissions_models.Permission{
				Name:        perm,
				Description: perm,
			}

			if result, err = pq.InsertPermission(
				context.Background(), permObj.Name, permObj.Description,
			); err != nil {
				panic(err)
			}

			permObj.ID = uint64(result)
			permissionIdsMap[uint64(i+1)] = append(
				permissionIdsMap[uint64(i+1)], permObj.ID,
			)
			permissionNamesMap[perm] = permObj.ID
		}

		var group = permissions_models.Group{
			Name:        fmt.Sprintf("group%d", i),
			Description: fmt.Sprintf("group%d", i),
		}
		_, err = pq.InsertGroup(
			context.Background(), group.Name, group.Description,
		)
		if err != nil {
			panic(err)
		}

		for _, permID := range permissionIdsMap[uint64(i+1)] {
			_, err = pq.InsertGroupPermission(
				context.Background(), uint64(i+1), permID,
			)
			if err != nil {
				panic(err)
			}
		}

		users = append(users, u)
	}

	var ctx = context.Background()
	for _, u := range users {
		if err := u.Save(ctx); err != nil {
			panic(err)
		}
	}
}

func buildUserRequest(user authentication.User) *http.Request {
	var request = httptest.NewRequest("GET", "/", nil)
	return request.WithContext(
		context.WithValue(
			request.Context(),
			context_user_key,
			user,
		),
	)
}

func TestBuildRequestUser(t *testing.T) {
	var requestWithUser = buildUserRequest(&users[0])
	var user = authentication.Retrieve(requestWithUser)
	if user == nil {
		t.Fatalf("expected user to be non-nil")
	}
}

func TestHasPermissions(t *testing.T) {
	// for userId, permNames := range permissionMap {
	// var request = httptest.NewRequest("GET", "/", nil)
	// request = request.WithContext(context.WithValue(
	// request.Context(), context_user_key, &models.User{
	// ID: userId,
	//
	// },
	// ))
	// if !permissions.HasPermission(userId, permNames...) {
	//
	// }
	// }

	t.Run("Success", func(t *testing.T) {
		for _, user := range users {
			var perms = permissionMap[user.ID]
			var request = buildUserRequest(&user)

			if !permissions.HasPermission(request, perms...) {
				t.Errorf("expected user %d to have permissions %v", user.ID, perms)
			}

			for _, perm := range perms {
				if user.IsAuthenticated() {
					if !permissions.HasPermission(request, perm) {
						t.Errorf("expected user %d to have permission %s", user.ID, perm)
					}
				} else {
					if permissions.HasPermission(request, perm) {
						t.Errorf("expected user %d to not have permission %s", user.ID, perm)
					}
				}
			}
		}
	})

	t.Run("Fail", func(t *testing.T) {
		for _, user := range users {
			var request = buildUserRequest(&user)
			var hasPerm = permissions.HasPermission(request, "doesnt_exist.add_user")
			if user.IsAdmin() && !hasPerm {
				t.Fatalf("expected user %d to have permission %s", user.ID, "doesnt_exist.add_user")
			}

			if hasPerm {
				t.Fatalf("expected user to not have permission %s: %+v", "doesnt_exist.add_user", user)
			}
		}
	})

	t.Run("NoPermissions", func(t *testing.T) {
		for _, user := range users {
			var request = buildUserRequest(&user)
			if !permissions.HasPermission(request) {
				t.Errorf("No permissions provided should allow all users")
			}
		}
	})

	t.Run("PermissionsForGroup", func(t *testing.T) {
		var mailAddr, _ = mail.ParseAddress("test_user@example.com")
		var user = &models.User{
			Username:   "test_user",
			Email:      (*models.Email)(mailAddr),
			IsLoggedIn: true,
			IsActive:   true,
		}
		auth.SetPassword(user, "password")

		var err error
		if err = user.Save(context.Background()); err != nil {
			t.Fatalf("error saving user: %v", err)
		}

		if user.ID == 0 {
			t.Fatalf("expected user to have an ID")
		}

		if !user.IsAuthenticated() {
			t.Fatalf("expected user to be authenticated")
		}

		var group = permissions_models.Group{
			Name:        "test_group",
			Description: "test_group",
		}

		var groupID int64
		if groupID, err = pq.InsertGroup(
			context.Background(), group.Name, group.Description,
		); err != nil {
			t.Fatalf("error inserting group: %v", err)
		}

		if _, err = pq.InsertUserGroup(
			context.Background(), user.ID, uint64(groupID),
		); err != nil {
			t.Fatalf("error inserting user group: %v", err)
		}

		var perm = permissions_models.Permission{
			Name:        "test_permission",
			Description: "test_permission",
		}

		var permID int64
		if permID, err = pq.InsertPermission(
			context.Background(), perm.Name, perm.Description,
		); err != nil {
			t.Fatalf("error inserting permission: %v", err)
		}

		if _, err = pq.InsertGroupPermission(
			context.Background(), uint64(groupID), uint64(permID),
		); err != nil {
			t.Fatalf("error inserting group permission: %v", err)
		}

		var request = buildUserRequest(user)
		if !permissions.HasPermission(request, perm.Name) {
			t.Fatalf("expected user to have permission %s", perm.Name)
		}

		// t.Run("TestInactiveUser", func(t *testing.T) {
		user.IsActive = false
		if permissions.HasPermission(request, perm.Name) {
			t.Fatalf("expected user to not have permission %s", perm.Name)
		}
		// })

		// t.Run("TestDeletedPermissionGroup", func(t *testing.T) {
		user.IsActive = true
		pq.DeleteGroupPermission(context.Background(), uint64(groupID), uint64(permID))
		if permissions.HasPermission(request, perm.Name) {
			t.Fatalf("expected user to not have permission %s", perm.Name)
		}
		// })

		// t.Run("TestAdminUser", func(t *testing.T) {
		user.IsAdministrator = true
		if !permissions.HasPermission(request, perm.Name) {
			t.Fatalf("expected user to have permission %s", perm.Name)
		}
		// })
	})
}
