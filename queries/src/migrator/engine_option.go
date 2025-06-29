package migrator

import (
	"io/fs"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/goldcrest"
	"github.com/elliotchance/orderedmap/v2"
)

type EngineOption func(*MigrationEngine)

func EngineOptionApps(apps ...string) EngineOption {
	var (
		appMap               = orderedmap.NewOrderedMap[string, django.AppConfig]()
		migrationDirectories = make(map[string]fs.FS)
	)

	if len(apps) == 0 {
		appMap = django.Global.Apps
		for head := appMap.Front(); head != nil; head = head.Next() {
			var app = head.Value
			var fs = getAppConfigFS(app)
			if fs != nil {
				migrationDirectories[head.Key] = fs
			}
			appMap.Set(head.Key, app)
		}

		goto retOption
	}

	for _, app := range apps {
		var appConfig = django.GetApp[django.AppConfig](app)
		var fs = getAppConfigFS(appConfig)
		if fs != nil {
			migrationDirectories[app] = fs
		}
		appMap.Set(app, appConfig)
	}

retOption:
	return func(e *MigrationEngine) {
		e.apps = appMap
		e.MigrationFilesystems = migrationDirectories
	}
}

func EngineOptionDirs(dirs ...string) EngineOption {
	return func(e *MigrationEngine) {
		e.Paths = dirs
	}
}

func getAppConfigFS(app django.AppConfig) fs.FS {
	if mgAppCnf, ok := app.(MigrationAppConfig); ok {
		var fs = mgAppCnf.GetMigrationFS()
		if fs != nil {
			return fs
		}
		return nil
	}

	// Try to get the migration FS from a hook
	var filesystems = goldcrest.Get[func() fs.FS](fileSystemHookName(
		app.Name(),
	))
	if len(filesystems) > 0 {
		var filesystemsList = make([]fs.FS, 0, len(filesystems))
		for _, fsFunc := range filesystems {
			filesystemsList = append(filesystemsList, fsFunc())
		}
		return filesystem.NewMultiFS(filesystemsList...)
	}

	return nil
}
