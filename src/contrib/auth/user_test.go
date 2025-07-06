package auth_test

import (
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/contrib/auth"
)

func TestUser(t *testing.T) {
	var u = &auth.User{
		Email:           drivers.MustParseEmail("test@example.com"),
		Username:        "testuser",
		FirstName:       "Test",
		LastName:        "User",
		IsAdministrator: false,
		IsActive:        true,
		IsLoggedIn:      false,
	}
	var user, err = auth.GetUserQuerySet().Create(u)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
		return
	}

	_, err = user.Groups.Objects().AddTarget(&auth.Group{
		Name: "Administrators",
	})
	if err != nil {
		t.Fatalf("Failed to add group to user: %v", err)
	}

	_, err = user.Permissions.Objects().AddTargets(
		&auth.Permission{
			Name:        "Can view users",
			Description: "Allows viewing of users",
		},
		&auth.Permission{
			Name:        "Can edit users",
			Description: "Allows editing of users",
		},
		&auth.Permission{
			Name:        "Can delete users",
			Description: "Allows deletion of users",
		},
	)
	if err != nil {
		t.Fatalf("Failed to add permissions to user: %v", err)
	}

	userRow, err := auth.GetUserQuerySet().Filter("ID", user.ID).First()
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	u = userRow.Object
	u.HasObjectPermission(nil, "Can view users", "Can edit users")
}
