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
	"iter"
	"maps"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	templateTranslationMatchers = map[string]func(tokens []string, colIdx, idx int) (string, string, int, int, error){
		"T": func(tokens []string, colIdx, idx int) (string, string, int, int, error) {
			if idx+1 >= len(tokens) || !(strings.HasPrefix(tokens[idx+1], `"`) || strings.HasPrefix(tokens[idx+1], "`")) {
				return "", "", 0, 0, fmt.Errorf("expected a string literal after T at index %d", idx)
			}
			// Remove outer quotes and unescape string
			unescaped, err := strconv.Unquote(tokens[idx+1])
			if err != nil {
				return "", "", 0, 0, fmt.Errorf("failed to unquote string %q: %w", tokens[idx+1], err)
			}

			return unescaped, "", colIdx, idx + 1, nil
		},
		"Translate": func(tokens []string, colIdx, idx int) (string, string, int, int, error) {
			switch {
			case idx+1 < len(tokens) && (strings.HasPrefix(tokens[idx+1], `"`) || strings.HasPrefix(tokens[idx+1], "`")):
				// Remove outer quotes and unescape string
				unescaped, err := strconv.Unquote(tokens[idx+1])
				if err != nil {
					return "", "", 0, 0, fmt.Errorf("failed to unquote string %q: %w", tokens[idx+1], err)
				}
				return unescaped, "", colIdx, idx + 1, nil
			case idx+2 < len(tokens) && (strings.HasPrefix(tokens[idx+2], `"`) || strings.HasPrefix(tokens[idx+2], "`")):
				// Remove outer quotes and unescape string
				unescaped, err := strconv.Unquote(tokens[idx+2])
				if err != nil {
					return "", "", 0, 0, fmt.Errorf("failed to unquote string %q: %w", tokens[idx+2], err)
				}
				return unescaped, "", colIdx, idx + 2, nil
			}

			return "", "", 0, 0, fmt.Errorf("expected string literals for Translate at indices %d and %d", idx+1, idx+2)
		},
		"P": func(tokens []string, colIdx, idx int) (string, string, int, int, error) {
			if idx+2 >= len(tokens) || !(strings.HasPrefix(tokens[idx+1], `"`) || strings.HasPrefix(tokens[idx+1], "`")) || !(strings.HasPrefix(tokens[idx+2], `"`) || strings.HasPrefix(tokens[idx+2], `'`)) {
				return "", "", 0, 0, fmt.Errorf("expected string literals for P at indices %d and %d", idx+1, idx+2)
			}

			singular, err := strconv.Unquote(tokens[idx+1])
			if err != nil {
				return "", "", 0, 0, fmt.Errorf("failed to unquote singular string %q: %w", tokens[idx+1], err)
			}
			plural, err := strconv.Unquote(tokens[idx+2])
			if err != nil {
				return "", "", 0, 0, fmt.Errorf("failed to unquote plural string %q: %w", tokens[idx+2], err)
			}
			return singular, plural, idx + 2, colIdx, nil
		},
	}

	goFileTranslationMatchers = map[string]func(call *ast.CallExpr, currentFunc string) (singular, plural *ast.BasicLit, err error){
		"S": func(call *ast.CallExpr, currentFunc string) (singular, plural *ast.BasicLit, err error) {
			if len(call.Args) == 0 {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected at least 1 argument for S, got %d", len(call.Args),
				)
			}
			singular, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected a string literal for S, got %s (%s)", reflect.TypeOf(call.Args[0]), stringCall(call),
				)
			}
			return singular, nil, nil
		},
		"T": func(call *ast.CallExpr, currentFunc string) (singular, plural *ast.BasicLit, err error) {
			if len(call.Args) < 2 {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected at least 2 arguments for T, got %d", len(call.Args),
				)
			}
			singular, ok := call.Args[1].(*ast.BasicLit)
			if !ok {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected a string literal for T, got %s (%s)", reflect.TypeOf(call.Args[1]), stringCall(call),
				)
			}
			return singular, nil, nil
		},
		"P": func(call *ast.CallExpr, currentFunc string) (singular, plural *ast.BasicLit, err error) {
			if len(call.Args) < 3 {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected at least 3 arguments for P, got %d", len(call.Args),
				)
			}
			singular, ok := call.Args[1].(*ast.BasicLit)
			if !ok {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"expected a string literal for P, got %s (%s)", reflect.TypeOf(call.Args[1]), stringCall(call),
				)
			}

			plural, ok = call.Args[2].(*ast.BasicLit)
			return singular, plural, errIfNotOk(ok, "expected a string literal for P")
		},
	}
)

func stringCall(call *ast.CallExpr) string {
	var buf = new(bytes.Buffer)
	if err := printer.Fprint(buf, token.NewFileSet(), call); err != nil {
		return fmt.Sprintf("error printing call: %v", err)
	}
	return buf.String()
}

type templateTranslation struct {
	singular   string
	plural     string
	regexMatch string
	col        int
}

func parseGoTemplateTCalls(template string, parseFuncs map[string]func(tokens []string, col, idx int) (string, string, int, int, error)) iter.Seq2[int, templateTranslation] {
	blockRegex := regexp.MustCompile(`\{\{((?:.|\n|\r\n|\t)*?)\}\}`)
	blocks := blockRegex.FindAllStringSubmatchIndex(template, -1)

	tokenRegex := regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_.]*|"(?:\\.|[^"\\])*"|[(){}:=|]`)

	return func(yield func(int, templateTranslation) bool) {
		for matchIdx, match := range blocks {
			blockStart := match[0]        // Start of the full {{...}} block
			blockContentStart := match[2] // Start of the inner block content
			blockContent := template[blockContentStart:match[3]]

			tokenPositions := tokenRegex.FindAllStringIndex(blockContent, -1)
			tokenStrings := tokenRegex.FindAllString(blockContent, -1)

			for i := 0; i < len(tokenStrings); i++ {
				tokenStart := tokenPositions[i][0]
				absoluteOffset := blockStart + 2 + tokenStart // 2 accounts for the opening `{{`

				fn, ok := parseFuncs[tokenStrings[i]]
				if !ok {
					continue
				}

				singular, plural, _, newI, err := fn(tokenStrings, absoluteOffset, i)
				if err != nil {
					logger.Warnf(
						"Skipping template translation in %q: %s: %v",
						template, fmt.Sprintf("Invalid T call at index %d: %s", i, tokenStrings[i]), err,
					)
					continue
				}

				if !yield(matchIdx+i, templateTranslation{
					singular:   singular,
					plural:     plural,
					regexMatch: template[match[0]:match[1]],
					col:        absoluteOffset + 1, // Convert to 1-based index
				}) {
					return
				}

				i = newI
			}
		}
	}
}

type templateTranslationsFinder struct {
	extensions []string
	matches    map[string]func(tokens []string, colIdx, idx int) (string, string, int, int, error)
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
		src, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		scanner := bufio.NewScanner(bytes.NewReader(src))
		lineNum := 0
		matchCount := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for _, t := range parseGoTemplateTCalls(line, f.matches) {
				if t.singular == "" {
					continue
				}

				var col = t.col + 1 // Convert to 1-based index
				var match = Translation{
					Path:   filepath.ToSlash(path),
					Line:   lineNum,
					Col:    col,
					Text:   t.singular,
					Plural: t.plural,
					Comment: fmt.Sprintf(
						"[TemplateFinder]:\t%s",
						t.regexMatch,
					),
				}

				matches = append(matches, match)
				matchCount++
			}
		}
	}

	return matches, nil
}

type goTranslationsFinder struct {
	packageAliases []string
	functions      map[string]func(call *ast.CallExpr, currentFunc string) (singular, plural *ast.BasicLit, err error)
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

		var transPackageImported bool
		var transPackageImport string
		for _, imp := range file.Imports {
			if imp.Path.Value == trans_package_path {
				transPackageImported = true
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

		if !transPackageImported && len(f.packageAliases) == 0 && file.Name.Name != "trans" {
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

			var funcName string
			var packageName string
			var singular, plural *ast.BasicLit
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if !slices.Contains(funcNames, fun.Sel.Name) {
					return true
				}

				xIdent, ok := fun.X.(*ast.Ident)
				if !ok {
					logger.Warnf("Skipping %s: expected an identifier for %s", path, fun.Sel.Name)
					return true
				}
				funcName = fun.Sel.Name
				packageName = xIdent.Name
			case *ast.Ident:
				if !slices.Contains(funcNames, fun.Name) {
					return true
				}
				funcName = fun.Name
				packageName = transPackageImport
			default:
				return true
			}

			singular, plural, err = f.functions[funcName](call, funcName)
			if err != nil {
				// logger.Warnf("Skipping %s: %v", path, err)
				return true
			}

			if singular.Kind != token.STRING || plural != nil && plural.Kind != token.STRING {
				logger.Warnf("Skipping %s:%v:%v: expected string argument for %s", path, funcName)
				return true
			}

			if file.Name.Name != transPackageImport && packageName != transPackageImport && !slices.Contains(f.packageAliases, packageName) {
				logger.Warnf("Skipping %s: %s.%s is not a valid translation package", path, packageName, funcName)
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
				Path:    filepath.ToSlash(path),
				Line:    pos.Line,
				Col:     pos.Column,
				Text:    singularUnquoted,
				Plural:  pluralUnquoted,
				Comment: fmt.Sprintf("[GoFileFinder]:\t%s", currentFuncName),
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
				Path:    filepath.ToSlash(filepath.Join(".models", app.Name(), cType.Model())),
				Line:    lineNum,
				Col:     col,
				Text:    cType.Model(),
				Comment: fmt.Sprintf("[ModelFinder]:\t%s", cType.ShortTypeName()),
			}

			matches = append(matches, match)

			var fieldDefs = model.FieldDefs()
			for i, field := range fieldDefs.Fields() {
				var fieldMatch = Translation{
					Path:    filepath.ToSlash(filepath.Join(".models", app.Name(), cType.Model(), "fields")),
					Line:    col,
					Col:     i,
					Text:    field.Label(context.Background()),
					Comment: fmt.Sprintf("[ModelFinder.Field]:\t%s.%s", cType.ShortTypeName(), field.Name()),
				}
				matches = append(matches, fieldMatch)
			}
		}
	}

	return matches, nil
}
