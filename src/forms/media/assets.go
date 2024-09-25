package media

import (
	"fmt"
	"html/template"
	"strings"
)

type CSS string

func (c CSS) String() string {
	return string(c)
}

func (c CSS) Render() template.HTML {
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, c))
}

type JS string

func (j JS) String() string {
	return string(j)
}

func (j JS) Render() template.HTML {
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, j))
}

type JSAsset struct {
	Type string
	URL  string
}

func (j *JSAsset) String() string {
	return j.URL
}

func (j *JSAsset) Render() template.HTML {
	var b = new(strings.Builder)
	b.WriteString(`<script`)
	if j.Type != "" {
		fmt.Fprintf(b, ` type="%s"`, j.Type)
	}
	fmt.Fprintf(b, ` src="%s"></script>`, j.URL)
	return template.HTML(b.String())
}
