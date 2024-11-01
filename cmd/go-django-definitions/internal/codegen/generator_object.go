package codegen

type TemplateObject struct {
	PackageName string
	SQLCVersion string
	Structs     []*Struct
	Choices     []*Choices
}

type Struct struct {
	Name   string
	Fields []Field
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
	Name    string
	Null    bool
	Blank   bool
	Primary bool
	Choices string
}
