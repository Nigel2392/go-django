package compare

import (
	"fmt"
	"html"
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
	var tagname = td.Tagname
	if tagname == "" {
		tagname = "span"
	}

	var htmlList = make([]string, 0, len(td.Changes))
	for _, change := range td.Changes {

		switch change.Type {
		case DIFF_EQUALS:
			htmlList = append(htmlList, html.EscapeString(attrs.ToString(change.Value)))
		case DIFF_ADDED:
			htmlList = append(htmlList, fmt.Sprintf(
				`<%s class="diff-added">%s</%s>`,
				tagname, html.EscapeString(attrs.ToString(change.Value)), tagname,
			))
		case DIFF_REMOVED:
			htmlList = append(htmlList, fmt.Sprintf(
				`<%s class="diff-removed">%s</%s>`,
				tagname, html.EscapeString(attrs.ToString(change.Value)), tagname,
			))
		}
	}

	return template.HTML(strings.Join(htmlList, td.Separator))
}
