package chocolaterie

import (
	"embed"
	"io/fs"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var assetFilesystem embed.FS

func NewAppConfig() django.AppConfig {
	var cfg = apps.NewAppConfig(
		"chocolaterie",
	)

	cfg.ModelObjects = []attrs.Definer{
		&Chocolate{},
		&ChocolateListPage{},
	}

	cfg.Routing = func(m mux.Multiplexer) {
		m.Handle(mux.GET, "/", mux.NewHandler(Index))
	}

	pages.SetRoutePrefix("/chocolaterie")

	cfg.Init = func(settings django.Settings) error {
		var tplFS, err = fs.Sub(assetFilesystem, "assets/templates")
		if err != nil {
			return err
		}

		staticFS, err := fs.Sub(assetFilesystem, "assets/static")
		if err != nil {
			return err
		}

		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("chocolaterie/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".css"),
				filesystem.MatchExt(".js"),
				filesystem.MatchExt(".png"),
				filesystem.MatchExt(".jpg"),
				filesystem.MatchExt(".jpeg"),
				filesystem.MatchExt(".svg"),
				filesystem.MatchExt(".gif"),
				filesystem.MatchExt(".ico"),
			),
		))

		tpl.Add(tpl.Config{
			AppName: "chocolaterie",
			FS:      tplFS,
			Bases: []string{
				"chocolaterie/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("chocolaterie/"),
				filesystem.MatchExt(".tmpl"),
			),
		})

		admin.RegisterApp(
			"chocolaterie",
			admin.AppOptions{
				AppLabel:  trans.S("Chocolaterie"),
				MenuLabel: trans.S("Chocolaterie"),
				MenuOrder: 500,
				MenuIcon: func() string {
					return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-fork-knife" viewBox="0 0 16 16">
  <path d="M13 .5c0-.276-.226-.506-.498-.465-1.703.257-2.94 2.012-3 8.462a.5.5 0 0 0 .498.5c.56.01 1 .13 1 1.003v5.5a.5.5 0 0 0 .5.5h1a.5.5 0 0 0 .5-.5zM4.25 0a.25.25 0 0 1 .25.25v5.122a.128.128 0 0 0 .256.006l.233-5.14A.25.25 0 0 1 5.24 0h.522a.25.25 0 0 1 .25.238l.233 5.14a.128.128 0 0 0 .256-.006V.25A.25.25 0 0 1 6.75 0h.29a.5.5 0 0 1 .498.458l.423 5.07a1.69 1.69 0 0 1-1.059 1.711l-.053.022a.92.92 0 0 0-.58.884L6.47 15a.971.971 0 1 1-1.942 0l.202-6.855a.92.92 0 0 0-.58-.884l-.053-.022a1.69 1.69 0 0 1-1.059-1.712L3.462.458A.5.5 0 0 1 3.96 0z"/>
</svg>`
				},
				// RegisterToAdminMenu: "admin:register_menu_item:settings",
				RegisterToAdminMenu: true,
				FullAdminMenu:       true,
			},
			admin.ModelOptions{
				Name:      "chocolate",
				Model:     &Chocolate{},
				MenuLabel: trans.S("Chocolates"),
				// RegisterToAdminMenu: "admin:register_menu_item:settings",
				RegisterToAdminMenu: true,
			},
		)

		pages.Register(&pages.PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				GetLabel:       trans.S("Chocolate List Page"),
				GetDescription: trans.S("A chocolate list page with a rich text editor."),
				ContentObject:  &ChocolateListPage{},
				Aliases: []string{
					"github.com/Nigel2392/go-django/example/chocolaterie/src/chocolaterie.ChocolateListPage",
				},
			},
			AddPanels: func(r *http.Request, page pages.Page) []admin.Panel {
				return []admin.Panel{
					admin.TitlePanel(
						admin.FieldPanel("Title"),
					),
					admin.RowPanel(
						admin.FieldPanel("UrlPath"),
						admin.FieldPanel("Slug"),
					),
					admin.FieldPanel("Description"),
				}
			},
			EditPanels: func(r *http.Request, page pages.Page) []admin.Panel {
				return []admin.Panel{
					admin.TitlePanel(
						admin.FieldPanel("Title"),
					),
					admin.RowPanel(
						admin.FieldPanel("UrlPath"),
						admin.FieldPanel("Slug"),
					),
					admin.FieldPanel("Description"),
					admin.FieldPanel("CreatedAt"),
					admin.FieldPanel("UpdatedAt"),
				}
			},
			ParentPageTypes: []string{},
			//GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			//	var row, err = queries.GetQuerySet(&BlogPage{}).Filter("PageID", id).First()
			//	if err != nil {
			//		return nil, errors.Wrapf(err, "failed to get blog page with ID %d", id)
			//	}
			//	*row.Object.PageNode = *ref
			//	return row.Object, nil
			//},
		})

		return nil
	}

	return cfg
}
