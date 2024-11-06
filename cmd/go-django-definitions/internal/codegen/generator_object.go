package codegen

import "slices"

type TemplateObject struct {
	PackageName string
	SQLCVersion string
	Structs     []*Struct
	Choices     []*Choices
	imports     map[string]struct{}
}

func (t *TemplateObject) AddImport(pkg string) {
	t.imports[pkg] = struct{}{}
}

func (t *TemplateObject) Imports() []string {
	var imports = make([]string, 0)
	for pkg := range t.imports {
		imports = append(imports, pkg)
	}
	slices.Sort(imports)
	return imports
}

type Struct struct {
	Name               string
	PluralName         string
	PrimaryField       Field
	PrimaryFieldColumn string
	TableName          string
	Fields             []Field
	InsertableFields   []Field
}

type Choices struct {
	Name    string
	Choices []Choice
}

type Choice struct {
	Name  string
	Value string
}

type Field struct {
	Name                         string
	ColumnName                   string
	Null                         bool
	Blank                        bool
	ReadOnly                     bool
	Primary                      bool
	Choices                      string
	RelatedObjectName            string
	RelatedObjectPackage         string
	RelatedObjectPackageAdressor string
	GoType                       string
	Widget                       string
}
