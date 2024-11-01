package codegen

import (
	"embed"
	"errors"
	"io"
	"path"
	"text/template"

	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen/plugin"
)

//go:embed templates/*
var templates embed.FS

const (
	GenerateDefinerTemplate = "definer.tmpl"
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
			Name:   c.opts.GoName(tbl.Rel.Name),
			Fields: make([]Field, 0, len(tbl.Columns)),
		}

		for i, col := range tbl.Columns {
			s.Fields = append(s.Fields, Field{
				Name:    c.opts.GoName(col.Name),
				Choices: choices[col.Type.Name],
				Null:    col.NotNull,
				Blank:   col.NotNull,
				Primary: i == 0,
			})
		}
		obj.Structs = append(obj.Structs, s)
	}

	return obj
}

func (c *CodeGenerator) Render(w io.Writer, name string, obj *TemplateObject) error {
	var tmpl, err = template.New(name).ParseFS(
		templates, path.Join("templates", name),
	)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, obj)
}
