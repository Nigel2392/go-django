package compare

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type DiffType string

const (
	DIFF_EQUALS  DiffType = "equals"
	DIFF_ADDED   DiffType = "added"
	DIFF_REMOVED DiffType = "removed"
)

type Differential struct {
	Type  DiffType
	Value any
}

type TextDiff struct {
	Changes   []Differential // list of changes
	Separator string         // e.g. " ", "\n", etc.
	Tagname   string         // e.g. "span", "div", etc.

}

func (td *TextDiff) HTML() template.HTML {
	var html = make([]string, 0, len(td.Changes))
	for _, change := range td.Changes {
		switch change.Type {
		case DIFF_EQUALS:
			html = append(html, template.HTMLEscapeString(attrs.ToString(change.Value)))
		case DIFF_ADDED:
			html = append(html, fmt.Sprintf(
				`<%s class="diff-added">%s</%s>`,
				td.Tagname, template.HTMLEscapeString(attrs.ToString(change.Value)), td.Tagname,
			))
		case DIFF_REMOVED:
			html = append(html, fmt.Sprintf(
				`<%s class="diff-removed">%s</%s>`,
				td.Tagname, template.HTMLEscapeString(attrs.ToString(change.Value)), td.Tagname,
			))
		}
	}

	return template.HTML(strings.Join(html, td.Separator))
}
