package auth_permissions

import (
	"database/sql"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/permissions"
)

func NewAppConfig() django.AppConfig {
	var cnf = apps.NewDBAppConfig("auth-permissions")
	cnf.Init = func(settings django.Settings, db *sql.DB) error {
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
