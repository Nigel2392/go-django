package codegen

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen/plugin"
	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/logger"
)

//go:embed templates/*
var templates embed.FS

const (
	GenerateDefinerTemplate = "attrs_definer.tmpl"
	GenerateModelsTemplate  = "models_interface.tmpl"
	GenerateAdminTemplate   = "admin_setup.tmpl"
)

func makeLabel(s string) string {
	return labelRegex.ReplaceAllString(s, "${1} ${2}")
}

var (
	Prefixes = map[string]string{
		GenerateDefinerTemplate: "godjango_definer",
		GenerateModelsTemplate:  "godjango_models",
		GenerateAdminTemplate:   "godjango_admin",
	}
	labelRegex = regexp.MustCompile(`([a-z])([A-Z])`)
	funcMap    = template.FuncMap{
		"label": func(s string) string {
			return makeLabel(s)
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

func colIsReadOnly(col *plugin.Column, commentMap map[string]string, readOnlyMap map[string]struct{}) bool {
	if _, ok := readOnlyMap[col.Name]; ok {
		logger.Logf("Field %s is read only\n", col.Name)
		return true
	}

	if commentMap != nil && slices.Contains(
		[]string{"1", "true", "yes", "y", "on"},
		strings.ToLower(commentMap["readonly"]),
	) {
		logger.Logf("Field %s is read only\n", col.Name)
		return true
	}
	return false
}

var parseCommentsRegex = regexp.MustCompile(`(\w+)\s*:\s*("[^"]*"|'[^']*'|[a-zA-Z0-9_\-\%=\./,]+)`)

func parseComments(comments string) map[string]string {
	var m = make(map[string]string)
	var matches = parseCommentsRegex.FindAllStringSubmatch(comments, -1)
	for _, match := range matches {
		m[match[1]] = match[2]
	}
	return m
}

func fmtPkgName(name string) string {
	name = strings.ToLower(name)
	var wasSpecial bool
	return strings.Map(func(r rune) rune {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if !wasSpecial {
				wasSpecial = true
				return '_'
			}
			return -1
		}
		wasSpecial = false
		return r
	}, name)
}

func packageInfo(objectPath string) (pkg, pkgAdressor, objName string) {
	var lastDot = strings.LastIndex(objectPath, ".")
	if lastDot != -1 {
		var lastSlash = strings.LastIndex(objectPath, "/")
		if lastSlash != -1 {
			pkg = objectPath[:lastDot]
			pkgAdressor = fmtPkgName(objectPath[lastSlash+1 : lastDot])
			objName = objectPath[lastDot+1:]
		} else {
			pkg = objectPath[:lastDot]
			pkgAdressor = fmtPkgName(objectPath[:lastDot])
			objName = objectPath[lastDot+1:]
		}
	} else {
		objName = objectPath
	}
	return
}

func (c *CodeGenerator) BuildTemplateObject(schema *plugin.Schema) *TemplateObject {
	var obj = &TemplateObject{
		PackageName: c.opts.PackageName,
		SQLCVersion: c.opts.req.SqlcVersion,
		Structs:     make([]*Struct, 0),
		Choices:     make([]*Choices, 0),
		imports:     make(map[string]Import),
	}

	logger.Logf("Building template object for schema %s\n", schema.Name)
	logger.Logf("Package name: %s\n", c.opts.PackageName)
	logger.Logf("SQLC Version: %s\n", c.opts.req.SqlcVersion)
	logger.Logf("Structs: %d\n", len(schema.Tables))
	logger.Logf("Enums: %d\n", len(schema.Enums))

	// Parse the schema enums and build the choices
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

	// Parse the schema tables and build the struct fields
	for _, tbl := range schema.Tables {
		var s = &Struct{
			Name: c.opts.GoName(
				c.opts.InflectSingular(tbl.Rel.Name),
			),
			PluralName:       c.opts.GoName(tbl.Rel.Name),
			TableName:        tbl.Rel.Name,
			Fields:           make([]Field, 0, len(tbl.Columns)),
			InsertableFields: make([]Field, 0, len(tbl.Columns)),
		}

		// Parse table comments for struct level directives
		var structCommentMap map[string]string
		if tbl.Comment != "" {
			structCommentMap = parseComments(tbl.Comment)
		}
		var readOnly, err = strconv.Unquote(
			structCommentMap["readonly"],
		)
		if err != nil {
			readOnly = structCommentMap["readonly"]
		}
		var readOnlyFields = strings.Split(readOnly, ",")
		var readOnlyMap = make(map[string]struct{})
		for _, f := range readOnlyFields {
			var f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			readOnlyMap[f] = struct{}{}
			logger.Logf("Field %s %d is read only\n", f, len(f))
		}

		logger.Logf("Building struct %s\n", s.Name)
		logger.Logf("Metadata: %+v\n", structCommentMap)
		logger.Logf("Read only fields: %+v\n", readOnlyFields)

		// Walk through the columns and build the struct fields
		for i, col := range tbl.Columns {
			var commentMap map[string]string
			if col.Comment != "" {
				commentMap = parseComments(col.Comment)
			}

			var f = Field{
				Name:       c.opts.GoName(col.Name),
				ColumnName: col.Name,
				Choices:    choices[col.Type.Name],
				Null:       col.NotNull,
				Blank:      col.NotNull,
				ReadOnly: colIsReadOnly(
					col, commentMap, readOnlyMap,
				),
				Primary: i == 0,
				GoType: goType(
					c, col,
				),
			}

			logger.Logf("Building field %s.%s\n", s.Name, col.Name)
			logger.Logf("Metadata: %+v\n", commentMap)

			if fk, ok := commentMap["fk"]; ok {
				var unquoted, err = strconv.Unquote(fk)
				if err != nil {
					unquoted = fk
				}
				var values = strings.SplitN(unquoted, "=", 2)
				if len(values) == 2 {
					var pkgPath, pkgAdressor, dotObject = packageInfo(
						values[1],
					)
					f.RelatedObjectPackage = pkgPath
					f.RelatedObjectPackageAdressor = pkgAdressor
					f.RelatedObjectName = dotObject

					obj.AddImport(Import{
						Alias:   pkgAdressor,
						Package: pkgPath,
					})

				} else {
					f.RelatedObjectName = c.opts.GoName(
						c.opts.InflectSingular(unquoted),
					)
				}
			}

			if label, ok := commentMap["label"]; ok {
				var unquoted, err = strconv.Unquote(label)
				if err != nil {
					unquoted = label
				}
				f.Label = unquoted
			}

			if helpText, ok := commentMap["helptext"]; ok {
				var unquoted, err = strconv.Unquote(helpText)
				if err != nil {
					unquoted = helpText
				}
				f.HelpText = unquoted
			}

			var widgetAttrs = make([]Attribute, 0)
			for k, v := range commentMap {
				if strings.HasPrefix(k, "form_") {
					var attr, err = strconv.Unquote(v)
					if err != nil {
						attr = v
					}
					widgetAttrs = append(
						widgetAttrs,
						Attribute{
							Key:   strings.TrimPrefix(k, "form_"),
							Value: attr,
						},
					)
				}
			}

			if len(widgetAttrs) > 0 {
				slices.SortStableFunc(widgetAttrs, func(i, j Attribute) int {
					if i.Key < j.Key {
						return -1
					}
					if i.Key > j.Key {
						return 1
					}
					return 0
				})

				f.WidgetAttrs = widgetAttrs
			}

			if !f.ReadOnly {
				s.InsertableFields = append(s.InsertableFields, f)
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
