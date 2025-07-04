package testsql

import (
	"embed"

	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
)

//go:embed auth_migrations/*
var auth_migrations embed.FS

//go:embed broad_migrations/*
var broad_migrations embed.FS

func NewAuthAppConfig() django.AppConfig {
	var cnf = apps.NewAppConfig("auth")
	var app = &migrator.MigratorAppConfig{
		AppConfig: cnf,
		MigrationFS: filesystem.Sub(
			auth_migrations, "auth_migrations/auth",
		),
	}
	cnf.ModelObjects = []attrs.Definer{
		&User{},
		&Profile{},
	}
	return app
	// return cnf
}

func NewTodoAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("todo")
	app.ModelObjects = []attrs.Definer{
		&Todo{},
	}
	return app
}

func NewBlogAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("blog")
	app.ModelObjects = []attrs.Definer{
		&BlogPost{},
		&BlogComment{},
	}
	return app
}

func NewBroadAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("broad")
	app.ModelObjects = []attrs.Definer{
		&Broad{},
	}
	return &migrator.MigratorAppConfig{
		AppConfig: app,
		MigrationFS: filesystem.Sub(
			broad_migrations, "broad_migrations/broad",
		),
	}
}
