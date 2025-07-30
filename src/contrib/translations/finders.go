package translations

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/dlclark/regexp2"
)

var (
	t_funcName                  = "T"
	p_funcName                  = "P"
	l_delim                     = `\{\{`
	r_delim                     = `\}\}`
	translationTemplateRegexStr = fmt.Sprintf(
		`%s\s*%s\s*"((?:[^"\\]|\\.)*)"(?:[a-zA-Z0-9]|[^a-zA-Z0-9%s])*%s`,
		l_delim, t_funcName, r_delim, r_delim,
	)
	translationTemplateRegexPluralStr = fmt.Sprintf(
		`%s\s*%s\s*"((?:[^"\\]|\\.)*)"\s*"((?:[^"\\]|\\.)*)"(?:[a-zA-Z0-9]|[^a-zA-Z0-9%s])*%s`,
		l_delim, p_funcName, r_delim, r_delim,
	)
	translationTemplatePipeRegexStr = fmt.Sprintf(
		`%s\s*(?:"((?:[^"\\]|\\.)+)"|(\w[\w\d\-_]*))\s*\|\s*%s\b[^}]*%s`,
		l_delim, t_funcName, r_delim,
	)
	translationTemplateRegex = regexp2.MustCompile(
		translationTemplateRegexStr,
		regexp2.RE2,
	)
	translationTemplateRegexPlural = regexp2.MustCompile(
		translationTemplateRegexPluralStr,
		regexp2.RE2,
	)
	translationTemplatePipeRegex = regexp2.MustCompile(
		translationTemplatePipeRegexStr,
		regexp2.RE2,
	)
)

type templateTranslationsFinder struct {
	extensions []string
}

func (f *templateTranslationsFinder) Find(fsys fs.FS) ([]Translation, error) {
	var paths []string

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		for _, ext := range f.extensions {
			if filepath.Ext(d.Name()) == "."+ext {
				paths = append(paths, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var closers []func() error
	var matches []Translation
	defer func() {
		for _, closer := range closers {
			if err := closer(); err != nil {
				logger.Errorf("Error closing file: %v", err)
			}
		}
	}()

	for _, path := range paths {
		file, err := fsys.Open(path)
		if err != nil {
			return nil, err
		}

		defer func() {
			if err := recover(); err != nil {
				closers = append(closers, file.Close)
				logger.Errorf("Error processing file %s: %v", path, err)
			}
		}()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		matchCount := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Capture stuff like {{ T "my-text" }} and {{ T "my-text" | html | myfunc }}
			// note: also captures {{ T "my-text" | T }} and {{ T "my-text" | html | myfunc }}
			var rexMatch, err = translationTemplateRegex.FindStringMatch(line)
			for rexMatch != nil && err == nil {
				var capture = rexMatch.Groups()[1].Captures[0].String()
				var col = rexMatch.Index + 1 // column is 1-based
				matches = append(matches, Translation{
					Path: path,
					Line: lineNum,
					Col:  col,
					Text: capture,
					Comment: fmt.Sprintf(
						"[TemplateFinder]: %s",
						rexMatch.String(),
					),
				})

				matchCount++
				rexMatch, err = translationTemplateRegex.FindNextMatch(rexMatch)
			}
			if err != nil {
				file.Close()
				return nil, err
			}

			// Capture stuff like {{ "my-text" | T }} and {{ "my-text" | T | html }} but not {{ "my-text" | html | T }}
			rexMatch, err = translationTemplatePipeRegex.FindStringMatch(line)
			for rexMatch != nil && err == nil {
				var capture = rexMatch.Groups()[1].Captures[0].String()
				var col = rexMatch.Index + 1 // column is 1-based
				matches = append(matches, Translation{
					Path: path,
					Line: lineNum,
					Col:  col,
					Text: capture,
					Comment: fmt.Sprintf(
						"[TemplateFinder]: %s",
						rexMatch.String(),
					),
				})

				matchCount++
				rexMatch, err = translationTemplatePipeRegex.FindNextMatch(rexMatch)
			}
			if err != nil {
				file.Close()
				return nil, err
			}

			// Capture stuff like {{ P "my-text" "my-text" var }} and {{ P "my-text" "my-text" var | html | myfunc }}
			rexMatch, err = translationTemplateRegexPlural.FindStringMatch(line)
			for rexMatch != nil && err == nil {
				var capture = rexMatch.Groups()[1].Captures[0].String()
				var plural = rexMatch.Groups()[2].Captures[0].String()
				var col = rexMatch.Index + 1 // column is 1-based
				matches = append(matches, Translation{
					Path:    path,
					Line:    lineNum,
					Col:     col,
					Text:    capture,
					Plural:  plural,
					Comment: fmt.Sprintf("[TemplateFinder]: %s", rexMatch.String()),
				})

				matchCount++
				rexMatch, err = translationTemplateRegexPlural.FindNextMatch(rexMatch)
			}
			if err != nil {
				file.Close()
				return nil, err
			}
		}

		file.Close()
	}

	return matches, nil
}

type goTranslationsFinder struct {
	packageAliases []string
}

func (f *goTranslationsFinder) Find(fsys fs.FS) ([]Translation, error) {
	var matches []Translation

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		src, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		var base = filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			// You can log but skip the file if broken
			logger.Warnf("Skipping broken Go file: %s", path)
			return nil
		}

		var currentFuncName string
		var funcEnd token.Pos
		ast.Inspect(file, func(n ast.Node) bool {

			if n != nil && n.End() > funcEnd {
				funcEnd = n.End()
				currentFuncName = ""
			}

			var call *ast.CallExpr
			switch n := n.(type) {
			case *ast.FuncDecl:

				var funcName = n.Name.Name // <-- function name here
				if n.Recv != nil && len(n.Recv.List) > 0 {
					// It's a method! Let's get the receiver type:
					// n.Recv.List[0].Type is an ast.Expr representing the receiver type.
					buf := new(bytes.Buffer)
					printer.Fprint(buf, token.NewFileSet(), n.Recv.List[0].Type)
					receiverType := buf.String() // e.g. "*MyStruct" or "MyStruct"
					funcName = receiverType + "." + funcName
				}
				currentFuncName = funcName
				return true

			case *ast.CallExpr:
				call = n

			default:
				return true
			}

			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "S" && selector.Sel.Name != "T" && selector.Sel.Name != "P") {
				return true
			}

			xIdent, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			var (
				ok1, ok2         bool = true, true
				singular, plural *ast.BasicLit
			)
			switch selector.Sel.Name {
			case "S":
				if len(call.Args) == 0 {
					logger.Warnf("Skipping %s: expected at least 1 argument for S, got none", path)
					return true
				}
				singular, ok1 = call.Args[0].(*ast.BasicLit)
			case "T":
				if len(call.Args) < 2 {
					logger.Warnf("Skipping %s: expected at least 2 arguments for T, got %d", path, len(call.Args))
					return true
				}
				singular, ok1 = call.Args[1].(*ast.BasicLit)
			case "P":
				if len(call.Args) < 3 {
					logger.Warnf("Skipping %s: expected at least 3 arguments for P, got %d", path, len(call.Args))
					return true
				}
				singular, ok1 = call.Args[1].(*ast.BasicLit)
				plural, ok2 = call.Args[2].(*ast.BasicLit)
			default:
				logger.Warnf("Skipping %s: unsupported selector %s", path, selector.Sel.Name)
				return true
			}

			if !ok1 || !ok2 || singular.Kind != token.STRING || plural != nil && plural.Kind != token.STRING {
				logger.Warnf("Skipping %s: expected string argument for %s", path, selector.Sel.Name)
				return true
			}

			if !slices.Contains(f.packageAliases, xIdent.Name) {
				return true
			}

			pos := fset.Position(singular.Pos())
			singularUnquoted, err := strconv.Unquote(singular.Value)
			if err != nil {
				singularUnquoted = singular.Value
			}

			var pluralUnquoted string
			if plural != nil {
				pluralUnquoted, err = strconv.Unquote(plural.Value)
				if err != nil {
					pluralUnquoted = plural.Value
				}
			}

			matches = append(matches, Translation{
				Path:    path,
				Line:    pos.Line,
				Col:     pos.Column,
				Text:    singularUnquoted,
				Plural:  pluralUnquoted,
				Comment: fmt.Sprintf("[GoFileFinder]: %s", currentFuncName),
			})

			return true
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

type godjangoModelsFinder struct {
}

func (f *godjangoModelsFinder) Find(fsys fs.FS) ([]Translation, error) {
	var apps = django.Global.Apps
	if apps == nil {
		return nil, nil
	}

	var matches []Translation
	var lineNum int
	for head := apps.Front(); head != nil; head = head.Next() {
		lineNum++

		var app = head.Value
		var models = app.Models()
		var col int

		for _, model := range models {
			col++

			var cType = contenttypes.NewContentType(model)
			var match = Translation{
				Path:    filepath.Join(".models", app.Name(), cType.Model()),
				Line:    lineNum,
				Col:     col,
				Text:    cType.Model(),
				Comment: fmt.Sprintf("[ModelFinder]: %s", cType.ShortTypeName()),
			}

			matches = append(matches, match)

			var fieldDefs = model.FieldDefs()
			for i, field := range fieldDefs.Fields() {
				var fieldMatch = Translation{
					Path:    filepath.Join(".models", app.Name(), cType.Model(), "fields"),
					Line:    col,
					Col:     i,
					Text:    field.Label(context.Background()),
					Comment: fmt.Sprintf("[ModelFinder.Field]: %s.%s", cType.ShortTypeName(), field.Name()),
				}
				matches = append(matches, fieldMatch)
			}
		}
	}

	return matches, nil
}
