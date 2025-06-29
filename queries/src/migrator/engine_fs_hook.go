package migrator

import (
	"fmt"
	"io/fs"

	"github.com/Nigel2392/goldcrest"
)

func RegisterFileSystem(app string, fSys fs.FS) int8 {
	if app == "" || fSys == nil {
		panic("app name and filesystem cannot be nil")
	}

	goldcrest.Register(
		fileSystemHookName(app), 0, func() fs.FS { return fSys },
	)

	return 0
}

func fileSystemHookName(app string) string {
	return fmt.Sprintf("migrations.fs.%s", app)
}
