package migrator

import (
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
)

var _ MigrationLog = &MigrationEngineConsoleLog{}

type MigrationEngineConsoleLog struct {
}

func (e *MigrationEngineConsoleLog) Log(action ActionType, file *MigrationFile, table *Changed[*ModelTable], column *Changed[*Column], index *Changed[*Index]) {
	var msg strings.Builder

	// Common prefix
	fmt.Fprintf(&msg, "%s/%s: ", file.AppName, file.ModelName)

	model := table.New.ModelName()
	tableName := table.New.TableName()

	switch action {
	case ActionCreateTable:
		fmt.Fprintf(&msg, "Create table %s for model %s", tableName, model)
	case ActionDropTable:
		fmt.Fprintf(&msg, "Drop table %s for model %s", tableName, model)
	case ActionRenameTable:
		fmt.Fprintf(&msg, "Rename table for model %s: %s → %s", model, table.Old.TableName(), tableName)
	case ActionAddIndex:
		fmt.Fprintf(&msg, "Add index %s on %s for model %s", index.New.Name(), tableName, model)
	case ActionDropIndex:
		fmt.Fprintf(&msg, "Drop index %s on %s for model %s", index.New.Name(), tableName, model)
	case ActionRenameIndex:
		fmt.Fprintf(&msg, "Rename index on %s for model %s: %s → %s", tableName, model, index.Old.Name(), index.New.Name())
	case ActionAddField:
		fmt.Fprintf(&msg, "Add field %s.%s on table %s", model, column.New.Name, tableName)
	case ActionAlterField:
		fmt.Fprintf(&msg, "Alter field %s on table %s for model %s", column.Old.Name, tableName, model)
	case ActionRemoveField:
		fmt.Fprintf(&msg, "Remove field %s on table %s for model %s", column.Old.Name, tableName, model)
	}

	logger.Info(msg.String())
}
