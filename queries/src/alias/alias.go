package alias

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
)

// generate an alias for fields.AliasField
func NewFieldAlias(tableAlias, alias string) string {
	if tableAlias == "" {
		return alias
	}
	return fmt.Sprintf("%s_%s", tableAlias, alias)
}

type Generator struct {
	Prefix  string
	counter int
	mapping map[string]string
}

func NewGenerator() *Generator {
	return &Generator{
		counter: 0,
		mapping: make(map[string]string),
	}
}

func (a *Generator) Clone() *Generator {
	return &Generator{
		Prefix:  a.Prefix,
		counter: a.counter,
		mapping: maps.Clone(a.mapping),
	}
}

// generate an alias for fields.AliasField
func (a *Generator) GetFieldAlias(tableAlias, alias string) string {
	//if tableAlias == "" {
	//	return alias
	//}
	//return fmt.Sprintf("%s_%s", tableAlias, alias)
	var aliasBuilder strings.Builder

	if tableAlias != "" {
		aliasBuilder.WriteString(tableAlias)
		aliasBuilder.WriteString("_")
	}

	aliasBuilder.WriteString(alias)

	return aliasBuilder.String()
}

func (a *Generator) GetTableAlias(currentTable string, chainorKey any) string {

	var key string
	switch v := chainorKey.(type) {
	case string:
		key = v
	case []string:
		key = strings.Join(v, ".")
	default:
		panic(fmt.Sprintf("unsupported type %T for (*Generator).GetAlias(...)", v))
	}

	var aliasBuilder strings.Builder
	if key == "" && currentTable != "" {
		aliasBuilder.WriteString(currentTable)
		return aliasBuilder.String()
	}

	if alias, ok := a.mapping[key]; ok {
		return alias
	}

	aliasBuilder.WriteString("T")

	if a.counter > 0 {
		aliasBuilder.WriteString(strconv.Itoa(
			a.counter,
		))
	}

	if currentTable != "" {
		aliasBuilder.WriteString("_")
		aliasBuilder.WriteString(currentTable)
	}

	var alias = aliasBuilder.String()
	a.mapping[key] = alias
	a.counter++
	return alias
}
