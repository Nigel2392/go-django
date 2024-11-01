package codegen

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen/plugin"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type CodeGeneratorOptions struct {
	Initialisms []string                `json:"initialisms"`
	Rename      map[string]string       `json:"rename"`
	PackageName string                  `json:"package"`
	OutFile     string                  `json:"out"`
	initialisms map[string]struct{}     `json:"-"`
	req         *plugin.GenerateRequest `json:"-"`
}

func (c *CodeGeneratorOptions) validate(req *plugin.GenerateRequest) error {
	if c.PackageName == "" {
		return errors.New("package name is required")
	}
	if c.OutFile == "" {
		c.OutFile = fmt.Sprintf("%s_definer.go", c.PackageName)
	}
	if c.Rename == nil {
		c.Rename = make(map[string]string)
	}
	if c.Initialisms == nil {
		c.Initialisms = []string{"id"}
	}
	c.initialisms = map[string]struct{}{}
	for _, initial := range c.Initialisms {
		c.initialisms[initial] = struct{}{}
	}
	c.req = req
	return nil
}

func (c *CodeGeneratorOptions) GoName(name string) string {
	if rename := c.Rename[name]; rename != "" {
		return rename
	}
	out := ""
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		if unicode.IsDigit(r) {
			return r
		}
		return rune('_')
	}, name)

	var caser = cases.Title(language.English)

	for _, p := range strings.Split(name, "_") {
		if _, found := c.initialisms[p]; found {
			out += strings.ToUpper(p)
		} else {
			out += caser.String(p)
		}
	}

	// If a name has a digit as its first char, prepand an underscore to make it a valid Go name.
	r, _ := utf8.DecodeRuneInString(out)
	if unicode.IsDigit(r) {
		return "_" + out
	}

	return out
}
