package auth_permissions

import (
	"github.com/Nigel2392/go-django-queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/permissions"
)

func NewAppConfig() django.AppConfig {
	var cnf = apps.NewDBAppConfig("auth-permissions")
	cnf.ModelObjects = []attrs.Definer{
		&permissions_models.Group{},
		&permissions_models.Permission{},
	}
	cnf.Init = func(settings django.Settings, db drivers.Database) error {
		var backend, err = NewQueries(db)
		if err != nil {
			return err
		}

		permissions.Tester = NewPermissionsBackend(
			backend,
		)

		err = backend.CreateTable()
		if err != nil {
			return err
		}

		//	admin.RegisterApp(
		//		cnf.Name(),
		//		admin.AppOptions{
		//			AppLabel:            trans.S("Permissions"),
		//			RegisterToAdminMenu: true,
		//		},
		//		admin.ModelOptions{
		//			Name:  "Groups",
		//			Model: &permissions_models.Group{},
		//		},
		//		admin.ModelOptions{
		//			Name:  "Permissions",
		//			Model: &permissions_models.Permission{},
		//		},
		//	)

		return nil
	}

	return cnf
}

func QuerySet() DBQuerier {
	if queries == nil {
		assert.Fail("queries for permissions app is nil")
	}
	return queries
}
