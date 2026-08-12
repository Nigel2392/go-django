package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type (
	ActionType int
)

func (a ActionType) String() string {
	return actionTypeToString[a]
}

func (a ActionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *ActionType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if v, ok := stringToActionType[s]; ok {
		*a = v
		return nil
	}
	return nil
}

const (
	ActionCreateTable ActionType = 1 << iota
	ActionDropTable
	ActionRenameTable
	ActionAddIndex
	ActionDropIndex
	ActionRenameIndex
	// ActionAlterUniqueTogether
	// ActionAlterIndexTogether
	ActionAddField
	ActionAlterField
	ActionRemoveField

	ActionExecGoCode
)

var actionTypeToString = map[ActionType]string{
	ActionCreateTable: "create_table",
	ActionDropTable:   "drop_table",
	ActionRenameTable: "rename_table",
	ActionAddIndex:    "add_index",
	ActionDropIndex:   "drop_index",
	ActionRenameIndex: "rename_index",
	// ActionAlterUniqueTogether: "alter_unique_together",
	// ActionAlterIndexTogether:  "alter_index_together",
	ActionAddField:    "add_field",
	ActionAlterField:  "alter_field",
	ActionRemoveField: "remove_field",
	ActionExecGoCode:  "exec",
}

var stringToActionType = map[string]ActionType{
	actionTypeToString[ActionCreateTable]: ActionCreateTable,
	actionTypeToString[ActionDropTable]:   ActionDropTable,
	actionTypeToString[ActionRenameTable]: ActionRenameTable,
	actionTypeToString[ActionAddIndex]:    ActionAddIndex,
	actionTypeToString[ActionDropIndex]:   ActionDropIndex,
	actionTypeToString[ActionRenameIndex]: ActionRenameIndex,
	// actionTypeToString[ActionAlterUniqueTogether]: ActionAlterUniqueTogether,
	// actionTypeToString[ActionAlterIndexTogether]:  ActionAlterIndexTogether,
	actionTypeToString[ActionAddField]:    ActionAddField,
	actionTypeToString[ActionAlterField]:  ActionAlterField,
	actionTypeToString[ActionRemoveField]: ActionRemoveField,
	actionTypeToString[ActionExecGoCode]:  ActionExecGoCode,
}

// Actions are kept track of to ensure a proper name can be generated for the migration file.
type MigrationAction struct {
	ActionType ActionType            `json:"action"`
	Table      *Changed[*ModelTable] `json:"table,omitempty"`
	Field      *Changed[*Column]     `json:"field,omitempty"`
	Index      *Changed[*Index]      `json:"index,omitempty"`
}

// map of model -> migration file -> func(*ModelTable)
var funcReg = make(map[reflect.Type]map[string]func(context.Context, *MigrationEngine, *ModelTable) error)

// Registers a function that [MigrationAction] can use if [MigrationAction.ActionType] == [ActionExecGoCode]
func RegisterMigrateFunc(model attrs.Definer, filename string, fn func(context.Context, *MigrationEngine, *ModelTable) error) {
	var rt = reflect.TypeOf(model)
	if rt == nil {
		panic("model is invalid")
	}

	if fn == nil {
		panic(fmt.Sprintf("%T: no func provided", model))
	}

	if filename == "" {
		panic(fmt.Sprintf("%T: no filename provided", model))
	}

	funcMap, ok := funcReg[rt]
	if !ok {
		funcMap = make(map[string]func(context.Context, *MigrationEngine, *ModelTable) error)
		funcReg[rt] = funcMap
	}

	_, ok = funcMap[filename]
	if ok {
		panic(fmt.Sprintf("%T: function is already registered", model))
	}

	funcMap[filename] = fn
}

func ExecMigrateFunc(ctx context.Context, engine *MigrationEngine, filename string, table *ModelTable) error {
	var rt = reflect.TypeOf(table.Object)
	if rt == nil {
		return errors.NilPointer.Wrap("model is invalid")
	}

	if filename == "" {
		return errors.ValueError.Wrapf("%T: no filename provided", table.Object)
	}

	funcMap, ok := funcReg[rt]
	if !ok {
		return errors.NotImplemented.Wrapf("No functions registered for model %T", table.Object)
	}

	fn, ok := funcMap[filename]
	if !ok || fn == nil {
		return errors.NotImplemented.Wrapf("No function registered for migration %q belonging to model %T: %v", filename, table.Object, funcMap)
	}

	return fn(ctx, engine, table)
}
