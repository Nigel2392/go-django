package codegen

type TemplateObject struct {
	PackageName string
	SQLCVersion string
	Structs     []*Struct
	Choices     []*Choices
}

type Struct struct {
	Name               string
	PluralName         string
	PrimaryField       Field
	PrimaryFieldColumn string
	TableName          string
	Fields             []Field
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
}
