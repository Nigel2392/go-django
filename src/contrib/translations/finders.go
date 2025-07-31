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
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/dlclark/regexp2"
)

var (
	l_delim_re = `\{\{`
	r_delim_re = `\}\}`

	templateTranslationMatchers = []templateTranslationMatcher{
		{
			regex: regexp2.MustCompile(
				fmt.Sprintf(
					`%s\s*%s\s*"((?:[^"\\]|\\.)*)"(?:[a-zA-Z0-9]|[^a-zA-Z0-9%s])*%s`,
					l_delim_re, "T", r_delim_re, r_delim_re,
				),
				regexp2.RE2,
			),
			exec: func(match *regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error) {
				var capture = match.Groups()[1].Captures[0].String()
				var col = match.Index + 1 // column is 1-based
				return capture, "", col, nil
			},
		},
		{
			regex: regexp2.MustCompile(
				fmt.Sprintf(
					`%s\s*%s\s*"((?:[^"\\]|\\.)*)"\s*"((?:[^"\\]|\\.)*)"(?:[a-zA-Z0-9]|[^a-zA-Z0-9%s])*%s`,
					l_delim_re, "P", r_delim_re, r_delim_re,
				),
				regexp2.RE2,
			),
			exec: func(match *regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error) {
				var capture = match.Groups()[1].Captures[0].String()
				var plural = match.Groups()[2].Captures[0].String()
				var col = match.Index + 1 // column is 1-based
				return capture, plural, col, nil
			},
		},
		{
			regex: regexp2.MustCompile(
				fmt.Sprintf(
					`%s\s*(?:"((?:[^"\\]|\\.)+)"|(\w[\w\d\-_]*))\s*\|\s*%s\b[^}]*%s`,
					l_delim_re, "T", r_delim_re,
				),
				regexp2.RE2,
			),
			exec: func(match *regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error) {
				var capture = match.Groups()[1].Captures[0].String()
				var col = match.Index + 1 // column is 1-based
				return capture, "", col, nil
			},
		},
	}

	goFileTranslationMatchers = map[string]func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error){
		"S": func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error) {
			if len(call.Args) == 0 {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected at least 1 argument for S, got %d", len(call.Args),
				)
			}
			singular, ok := call.Args[0].(*ast.BasicLit)
			return singular, nil, errIfNotOk(ok, "expected a string literal for S")
		},
		"T": func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error) {
			if len(call.Args) < 2 {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected at least 2 arguments for T, got %d", len(call.Args),
				)
			}
			singular, ok := call.Args[1].(*ast.BasicLit)
			return singular, nil, errIfNotOk(ok, "expected a string literal for T")
		},
		"P": func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error) {
			if len(call.Args) < 3 {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected at least 3 arguments for P, got %d", len(call.Args),
				)
			}
			singular, ok := call.Args[1].(*ast.BasicLit)
			if !ok {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected a string literal for P, got %s", reflect.TypeOf(call.Args[1]),
				)
			}

			plural, ok = call.Args[2].(*ast.BasicLit)
			return singular, plural, errIfNotOk(ok, "expected a string literal for P")
		},
	}
)

type templateTranslationMatcher struct {
	regex *regexp2.Regexp
	exec  func(*regexp2.Match) (trans.Untranslated, trans.Untranslated, int, error)
}

type templateTranslationsFinder struct {
	extensions []string
	matches    []templateTranslationMatcher
}

func (f *templateTranslationsFinder) Find(fsys fs.FS) ([]Translation, error) {
	var paths []string

	if f.matches == nil {
		f.matches = templateTranslationMatchers
	}

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

			for _, matcher := range f.matches {
				rexMatch, err := matcher.regex.FindStringMatch(line)
				if err != nil {
					file.Close()
					return nil, err
				}

				for rexMatch != nil && err == nil {
					singular, plural, col, err := matcher.exec(rexMatch)
					if err != nil {
						file.Close()
						return nil, err
					}

					matches = append(matches, Translation{
						Path:   path,
						Line:   lineNum,
						Col:    col,
						Text:   singular,
						Plural: plural,
						Comment: fmt.Sprintf(
							"[TemplateFinder]: %s",
							rexMatch.String(),
						),
					})

					matchCount++
					rexMatch, err = matcher.regex.FindNextMatch(rexMatch)
				}
				if err != nil {
					file.Close()
					return nil, err
				}
			}
		}

		file.Close()
	}

	return matches, nil
}

type goTranslationsFinder struct {
	packageAliases []string
	functions      map[string]func(call *ast.CallExpr, xIdent *ast.Ident, currentFunc string) (singular, plural *ast.BasicLit, err error)
}

func (f *goTranslationsFinder) Find(fsys fs.FS) ([]Translation, error) {
	var matches []Translation

	if f.functions == nil {
		f.functions = goFileTranslationMatchers
	}

	var funcNames = slices.Collect(maps.Keys(f.functions))
	var trans_package_path = fmt.Sprintf("\"%s\"", trans.PACKAGE_PATH)
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

		var transPackageImport string
		for _, imp := range file.Imports {
			if imp.Path.Value == trans_package_path {
				if imp.Name != nil {
					if imp.Name.Name == "." {
						transPackageImport = ""
					} else {
						transPackageImport = imp.Name.Name
					}
				} else {
					transPackageImport = "trans"
				}
				break
			}
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
			if !ok || !slices.Contains(funcNames, selector.Sel.Name) {
				return true
			}

			xIdent, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}

			singular, plural, err := f.functions[selector.Sel.Name](call, xIdent, currentFuncName)
			if err != nil {
				logger.Warnf("Skipping %s: %v", path, err)
				return true
			}

			if !ok || singular.Kind != token.STRING || plural != nil && plural.Kind != token.STRING {
				logger.Warnf("Skipping %s: expected string argument for %s", path, selector.Sel.Name)
				return true
			}

			if xIdent.Name != transPackageImport && !slices.Contains(f.packageAliases, xIdent.Name) {
				logger.Warnf("Skipping %s: %s.%s is not a valid translation package", path, xIdent.Name, selector.Sel.Name)
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
