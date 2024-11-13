package codegen

import (
	"fmt"
	"slices"
)

type Import struct {
	Package string
	Alias   string
}

func (i *Import) String() string {
	if i.Alias != "" {
		return fmt.Sprintf("%s \"%s\"", i.Alias, i.Package)
	}
	return fmt.Sprintf("\"%s\"", i.Package)
}

type TemplateObject struct {
	PackageName string
	SQLCVersion string
	Structs     []*Struct
	Choices     []*Choices
	imports     map[string]Import
}

func (t *TemplateObject) AddImport(imp Import) {
	if t.imports == nil {
		t.imports = make(map[string]Import)
	}
	t.imports[imp.Package] = imp
}

func (t *TemplateObject) Imports() []Import {
	var imports = make([]Import, 0)
	for _, imp := range t.imports {
		imports = append(imports, imp)
	}
	slices.SortStableFunc(imports, func(i, j Import) int {
		if i.Package < j.Package {
			return -1
		}
		if i.Package > j.Package {
			return 1
		}
		return 0
	})
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

type Attribute struct {
	Key   string
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
	Label                        string
	HelpText                     string
	RelatedObjectName            string
	RelatedObjectPackage         string
	RelatedObjectPackageAdressor string
	WidgetAttrs                  []Attribute
	GoType                       string
}

func (f *Field) IsRel() bool {
	return f.RelatedObjectName != ""
}

func (f *Field) GetLabel() string {
	if f.Label != "" {
		return f.Label
	}
	return makeLabel(f.Name)
}

func (f *Field) GetHelpText() string {
	if f.HelpText != "" {
		return f.HelpText
	}
	return ""
}
