package migrator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var suffixWithoutDot = strings.TrimPrefix(MIGRATION_FILE_SUFFIX, ".")

func parseMigrationFileName(n string) (orderNum int, name string, err error) {
	// The migration file name is expected to be in the format <order>_<name>.sql
	// where <order> is an integer and <name> is the name of the migration.
	// For example: 0001_create_users_table.sql

	var parts = strings.SplitN(n, ".", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid migration file name %q", n)
	}

	if !strings.HasSuffix(parts[1], suffixWithoutDot) {
		return 0, "", fmt.Errorf("invalid migration file name %q, expected suffix %q", n, MIGRATION_FILE_SUFFIX)
	}

	parts = strings.SplitN(parts[0], "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid migration file name %q, expected format <order>_<file_name>", n)
	}

	orderNum, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", errors.Wrapf(err, "invalid order number %q", parts[0])
	}

	name = parts[1]
	return orderNum, name, nil
}

func generateMigrationFileName(mig *MigrationFile) string {
	// The migration file name is expected to be in the format <order>_<name>.migration
	// where <order> is an integer and <name> is the name of the migration.
	// For example: 0001_create_users_table.migration
	//var orderStr = fmt.Sprintf("%04d", orderNum)
	//return fmt.Sprintf("%s_%s%s", orderStr, name, MIGRATION_FILE_SUFFIX)

	var orderStr = fmt.Sprintf("%04d_", mig.Order)
	var sb = strings.Builder{}
	if len(mig.Actions) == 0 {
		return fmt.Sprintf(
			"%s%s%s",
			orderStr,
			"auto_generated",
			MIGRATION_FILE_SUFFIX,
		)
	}

	sb.WriteString(orderStr)
	var action = mig.Actions[0]
	switch action.ActionType {
	case ActionCreateTable:
		sb.WriteString("create_table")
	case ActionDropTable:
		sb.WriteString("drop_table")
	case ActionRenameTable:
		sb.WriteString("rename_table_")
		sb.WriteString(action.Table.Old.Table)
		sb.WriteString("_to_")
		sb.WriteString(action.Table.New.Table)
	case ActionAddIndex:
		sb.WriteString("add_idx_")
		sb.WriteString(action.Index.New.Name())
	case ActionDropIndex:
		sb.WriteString("drop_idx_")
		sb.WriteString(action.Table.New.Table)
		sb.WriteString("_on_")
		sb.WriteString(action.Index.Old.Name())
	case ActionRenameIndex:
		sb.WriteString("rename_idx_")
		sb.WriteString(action.Index.Old.Name())
		sb.WriteString("_to_")
		sb.WriteString(action.Index.New.Name())
	// case ActionAlterUniqueTogether:

	// case ActionAlterIndexTogether:

	case ActionAddField:
		sb.WriteString("add_field_")
		sb.WriteString(action.Field.New.Column)
	case ActionAlterField:
		sb.WriteString("alter_field_")
		sb.WriteString(action.Field.New.Column)
	case ActionRemoveField:
		sb.WriteString("remove_field_")
		sb.WriteString(action.Field.Old.Column)
	}

	if len(mig.Actions) > 1 {
		sb.WriteString("_and_")
		sb.WriteString(fmt.Sprintf("%d_more", len(mig.Actions)-1))
	}

	sb.WriteString(MIGRATION_FILE_SUFFIX)

	return sb.String()
}

var (
	timeTyp = reflect.TypeOf(time.Time{})
)

func EqualDefaultValue(a, b any) bool {

	var cDefault = reflect.ValueOf(a)
	var otherDefault = reflect.ValueOf(b)
	if cDefault.IsValid() != otherDefault.IsValid() {
		var (
			aIsZero bool
			bIsZero bool
		)

		if cDefault.IsValid() && cDefault.Kind() == reflect.Ptr && (cDefault.IsNil() || cDefault.Elem().IsZero()) ||
			cDefault.IsValid() && cDefault.Kind() != reflect.Ptr && cDefault.IsZero() ||
			!cDefault.IsValid() {
			aIsZero = true
		}

		if otherDefault.IsValid() && otherDefault.Kind() == reflect.Ptr && (otherDefault.IsNil() || otherDefault.Elem().IsZero()) ||
			otherDefault.IsValid() && otherDefault.Kind() != reflect.Ptr && otherDefault.IsZero() ||
			!otherDefault.IsValid() {
			bIsZero = true
		}

		if aIsZero != bIsZero {
			return false
		}

		return true
	}

	if cDefault.Type() == timeTyp && otherDefault.Type() == timeTyp {
		return cDefault.Interface().(time.Time).Equal(otherDefault.Interface().(time.Time))
	}

	if cDefault.IsValid() && otherDefault.IsValid() {
		if otherDefault.Type() != cDefault.Type() && otherDefault.Type().ConvertibleTo(cDefault.Type()) {
			otherDefault = otherDefault.Convert(cDefault.Type())
		}
	}

	if cDefault.Kind() != reflect.Ptr && cDefault.IsZero() != otherDefault.IsZero() ||
		cDefault.Kind() == reflect.Ptr && (cDefault.IsNil() != otherDefault.IsNil() || cDefault.Elem().IsZero() != otherDefault.Elem().IsZero()) {
		return false
	} else if cDefault.Kind() != reflect.Ptr && !cDefault.IsZero() ||
		cDefault.Kind() == reflect.Ptr && !cDefault.IsNil() {
		if !reflect.DeepEqual(cDefault.Interface(), otherDefault.Interface()) {
			return false
		}
	}

	return true
}
