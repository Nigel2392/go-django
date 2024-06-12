package media

import (
	"fmt"
	"html/template"
	"strings"
)

type CSSAsset struct {
	URL string
}

func (c *CSSAsset) String() string {
	return c.URL
}

func (c *CSSAsset) Render() template.HTML {
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, c.URL))
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
