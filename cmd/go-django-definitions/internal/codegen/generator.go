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
	GenerateDefinerTemplate = "definer.tmpl"
)

var (
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
			TableName: tbl.Rel.Name,
			Fields:    make([]Field, 0, len(tbl.Columns)),
		}

		if len(tbl.Columns) > 0 {
			s.PrimaryField = c.opts.GoName(tbl.Columns[0].Name)
			s.PrimaryFieldColumn = tbl.Columns[0].Name
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
