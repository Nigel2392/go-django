package migrator

import "encoding/json"

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
	ActionCreateTable ActionType = iota + 1
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
}

// Actions are kept track of to ensure a proper name can be generated for the migration file.
type MigrationAction struct {
	ActionType ActionType            `json:"action"`
	Table      *Changed[*ModelTable] `json:"table,omitempty"`
	Field      *Changed[*Column]     `json:"field,omitempty"`
	Index      *Changed[*Index]      `json:"index,omitempty"`
}
