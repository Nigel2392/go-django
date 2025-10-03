package forms

import (
	"context"
	"embed"
	"fmt"
	"html"
	"io"
	"io/fs"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
)

//go:embed assets/**
var formAssets embed.FS

func initTemplateLibrary() {
	var templates, err = fs.Sub(formAssets, "assets/templates")
	assert.True(err == nil, "failed to get form templates")

	static, err := fs.Sub(formAssets, "assets/static")
	assert.True(err == nil, "failed to get form static files")

	var templateConfig = tpl.Config{
		AppName: "forms",
		FS:      templates,
		Bases:   []string{},
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("forms/widgets/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".html"),
			),
		),
	}

	for _, hook := range goldcrest.Get[FormTemplateFSHook](FormTemplateFSHookName) {
		templateConfig.FS = hook(templateConfig.FS, &templateConfig)
	}

	for _, hook := range goldcrest.Get[FormTemplateStaticHook](FormTemplateStaticHookName) {
		static = hook(static)
	}

	tpl.Add(templateConfig)
	staticfiles.AddFS(static, filesystem.MatchPrefix("forms/"))
}

type (
	FormTemplateFSHook     func(fSys fs.FS, cnf *tpl.Config) fs.FS
	FormTemplateStaticHook func(fSys fs.FS) fs.FS
)

const (
	FormTemplateFSHookName     = "forms.TemplateFSHook"
	FormTemplateStaticHookName = "forms.TemplateStaticHook"
)

func init() {
	tpl.FirstRender().Listen(func(s signals.Signal[*tpl.TemplateRenderer], tr *tpl.TemplateRenderer) error {
		initTemplateLibrary()
		return nil
	})
}

type defaultRenderer struct{}

func (r *defaultRenderer) RenderAsP(w io.Writer, c context.Context, form BoundForm) error {
	for _, field := range form.Fields() {
		w.Write([]byte("<p>"))
		w.Write([]byte(field.Label()))
		w.Write([]byte("\n"))
		w.Write([]byte(field.HelpText()))
		w.Write([]byte("\n"))
		w.Write([]byte(field.Field()))
		w.Write([]byte("</p>"))
	}
	return nil
}

func (r *defaultRenderer) RenderAsUL(w io.Writer, c context.Context, form BoundForm) error {
	w.Write([]byte("<ul>\n"))
	for _, field := range form.Fields() {
		w.Write([]byte("\t<li>"))
		w.Write([]byte(field.Label()))
		w.Write([]byte("</li>\n"))

		w.Write([]byte("\t<li>"))
		w.Write([]byte(field.HelpText()))
		w.Write([]byte("</li>\n"))

		w.Write([]byte("\t<li>"))
		w.Write([]byte(field.Field()))
		w.Write([]byte("</li>\n"))

	}
	w.Write([]byte("</ul>"))
	return nil
}

func (r *defaultRenderer) RenderAsTable(w io.Writer, c context.Context, form BoundForm) error {
	w.Write([]byte("<table>"))
	for _, field := range form.Fields() {
		w.Write([]byte("<tr>"))
		w.Write([]byte("<td>"))
		w.Write([]byte(field.Label()))
		w.Write([]byte("</td>"))
		w.Write([]byte("<td>"))
		w.Write([]byte(field.HelpText()))
		w.Write([]byte(field.Field()))
		w.Write([]byte("</td>"))
		w.Write([]byte("</tr>"))
	}
	w.Write([]byte("</table>"))
	return nil
}

func (r *defaultRenderer) RenderField(w io.Writer, c context.Context, field BoundField, id string, name string, value interface{}, errors []error, attrs map[string]string, widgetCtx ctx.Context) (err error) {
	if err = r.RenderFieldLabel(w, c, field, id, name); err != nil {
		return err
	}
	if err = r.RenderFieldWidget(w, c, field, id, name, value, attrs, errors, widgetCtx); err != nil {
		return err
	}
	if err = r.RenderFieldHelpText(w, c, field, id, name); err != nil {
		return err
	}
	return nil
}

func (r *defaultRenderer) RenderFieldLabel(w io.Writer, ctx context.Context, field BoundField, id string, name string) error {
	var fld = field.Input()
	var labelText = fld.Label(ctx)
	fmt.Fprintf(w,
		"<label for=\"%s\">%s</label>",
		id, html.EscapeString(labelText),
	)
	return nil
}

func (r *defaultRenderer) RenderFieldHelpText(w io.Writer, ctx context.Context, field BoundField, id string, name string) error {
	var fld = field.Input()
	var helpText = fld.HelpText(ctx)
	if helpText == "" {
		return nil
	}

	w.Write([]byte(html.EscapeString(helpText)))
	return nil
}

func (r *defaultRenderer) RenderFieldWidget(w io.Writer, c context.Context, field BoundField, id string, name string, value interface{}, attrs map[string]string, errors []error, widgetCtx ctx.Context) error {
	return field.Widget().RenderWithErrors(
		c, w, id, name, value, errors, attrs, widgetCtx,
	)
}
