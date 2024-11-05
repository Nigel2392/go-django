package codegen

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"text/template"

	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen/plugin"
)

//go:embed templates/*
var templates embed.FS

const (
	GenerateDefinerTemplate = "attrs_definer.tmpl"
	GenerateModelsTemplate  = "models_interface.tmpl"
	GenerateAdminTemplate   = "admin_setup.tmpl"
)

var (
	Prefixes = map[string]string{
		GenerateDefinerTemplate: "godjango_definer",
		GenerateModelsTemplate:  "godjango_models",
		GenerateAdminTemplate:   "godjango_admin",
	}
	labelRegex = regexp.MustCompile(`([a-z])([A-Z])`)
	funcMap    = template.FuncMap{
		"label": func(s string) string {
			return labelRegex.ReplaceAllString(s, "${1} ${2}")
		},
	}
)

type CodeGenerator struct {
	opts *CodeGeneratorOptions
}

func New(req *plugin.GenerateRequest, opts *CodeGeneratorOptions) (*CodeGenerator, error) {
	if opts == nil {
		return nil, errors.New("options are required")
	}

	if err := opts.validate(req); err != nil {
		return nil, err
	}

	if opts.Rename == nil {
		opts.Rename = map[string]string{}
	}

	return &CodeGenerator{
		opts: opts,
	}, nil
}

func colIsReadOnly(i int, col *plugin.Column) bool {
	if i == 0 {
		return true
	}
	if col.Name == "created_at" || col.Name == "updated_at" {
		return true
	}
	return false
}

var parseCommentsRegex = regexp.MustCompile(`(\w+)\s*:\s*(\w+)`)

func parseComments(comments string) map[string]string {
	var m = make(map[string]string)
	var matches = parseCommentsRegex.FindAllStringSubmatch(comments, -1)
	for _, match := range matches {
		m[match[1]] = match[2]
	}
	return m
}

func (c *CodeGenerator) BuildTemplateObject(schema *plugin.Schema) *TemplateObject {
	var obj = &TemplateObject{
		PackageName: c.opts.PackageName,
		SQLCVersion: c.opts.req.SqlcVersion,
		Structs:     make([]*Struct, 0),
		Choices:     make([]*Choices, 0),
	}

	var choices = make(map[string]string)
	for _, enum := range schema.Enums {
		var chs = &Choices{
			Name:    c.opts.GoName(enum.Name),
			Choices: make([]Choice, 0),
		}
		for _, val := range enum.Vals {
			chs.Choices = append(chs.Choices, Choice{
				Name:  c.opts.GoName(val),
				Value: val,
			})
		}
		obj.Choices = append(obj.Choices, chs)
		choices[enum.Name] = chs.Name
	}

	for _, tbl := range schema.Tables {
		var s = &Struct{
			Name: c.opts.GoName(
				c.opts.InflectSingular(tbl.Rel.Name),
			),
			PluralName: c.opts.GoName(tbl.Rel.Name),
			TableName:  tbl.Rel.Name,
			Fields:     make([]Field, 0, len(tbl.Columns)),
		}

		for i, col := range tbl.Columns {
			var f = Field{
				Name:       c.opts.GoName(col.Name),
				ColumnName: col.Name,
				Choices:    choices[col.Type.Name],
				Null:       col.NotNull,
				Blank:      col.NotNull,
				ReadOnly:   colIsReadOnly(i, col),
				Primary:    i == 0,
				GoType:     dbType(c, col),
			}
			var commentMap map[string]string
			if col.Comment != "" {
				commentMap = parseComments(col.Comment)
			}

			if fk, ok := commentMap["fk"]; ok {
				f.RelatedObjectName = c.opts.GoName(
					c.opts.InflectSingular(fk),
				)
			}

			s.Fields = append(s.Fields, f)
		}

		if len(s.Fields) > 0 {
			s.PrimaryField = s.Fields[0]
			s.PrimaryFieldColumn = s.Fields[0].ColumnName
		}

		obj.Structs = append(obj.Structs, s)
	}

	return obj
}

func (c *CodeGenerator) Render(w io.Writer, name string, obj *TemplateObject) error {
	var tmpl = template.New(name)
	funcMap["placeholder"] = func(iter ...int) string {
		switch c.opts.req.Settings.Engine {
		case "postgres":
			if len(iter) == 0 {
				return "?"
			}
			return fmt.Sprintf("$%d", iter[0])
		default:
			return "?"
		}
	}
	tmpl = tmpl.Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(
		templates, path.Join("templates", name),
	)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, obj)
}

func dbType(c *CodeGenerator, col *plugin.Column) string {
	notNull := col.NotNull || col.IsArray
	unsigned := col.Unsigned
	columnType := col.Type.Name
	switch columnType {

	case "varchar", "text", "char", "tinytext", "mediumtext", "longtext":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "tinyint":
		if col.Length == 1 {
			if notNull {
				return "bool"
			}
			return "sql.NullBool"
		} else {
			if notNull {
				if unsigned {
					return "uint8"
				}
				return "int8"
			}
			// The database/sql package does not have a sql.NullInt8 type, so we
			// use the smallest type they have which is NullInt16
			return "sql.NullInt16"
		}

	case "year":
		if notNull {
			return "int16"
		}
		return "sql.NullInt16"

	case "smallint":
		if notNull {
			if unsigned {
				return "uint16"
			}
			return "int16"
		}
		return "sql.NullInt16"

	case "int", "integer", "mediumint":
		if notNull {
			if unsigned {
				return "uint32"
			}
			return "int32"
		}
		return "sql.NullInt32"

	case "bigint":
		if notNull {
			if unsigned {
				return "uint64"
			}
			return "int64"
		}
		return "sql.NullInt64"

	case "blob", "binary", "varbinary", "tinyblob", "mediumblob", "longblob":
		if notNull {
			return "[]byte"
		}
		return "sql.NullString"

	case "double", "double precision", "real", "float":
		if notNull {
			return "float64"
		}
		return "sql.NullFloat64"

	case "decimal", "dec", "fixed":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "enum":
		// TODO: Proper Enum support
		return "string"

	case "date", "timestamp", "datetime", "time":
		if notNull {
			return "time.Time"
		}
		return "sql.NullTime"

	case "boolean", "bool":
		if notNull {
			return "bool"
		}
		return "sql.NullBool"

	case "json":
		return "json.RawMessage"

	case "any":
		return "interface{}"

	default:
		for _, schema := range c.opts.req.Catalog.Schemas {
			for _, enum := range schema.Enums {
				if enum.Name == columnType {
					if notNull {
						if schema.Name == c.opts.req.Catalog.DefaultSchema {
							return c.opts.GoName(enum.Name)
						}
						return c.opts.GoName(schema.Name + "_" + enum.Name)
					} else {
						if schema.Name == c.opts.req.Catalog.DefaultSchema {
							return "Null" + c.opts.GoName(enum.Name)
						}
						return "Null" + c.opts.GoName(schema.Name+"_"+enum.Name)
					}
				}
			}
		}
		return "interface{}"
	}
}
