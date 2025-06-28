package migrator

import (
	"fmt"
	"io/fs"

	"github.com/Nigel2392/goldcrest"
)

func RegisterFileSystem(app string, fs fs.FS) int8 {
	if app == "" || fs == nil {
		panic("app name and filesystem cannot be nil")
	}

	goldcrest.Register(
		fileSystemHookName(app), 0, fs,
	)

	return 0
}

func fileSystemHookName(app string) string {
	return fmt.Sprintf("migrations.fs.%s", app)
}
