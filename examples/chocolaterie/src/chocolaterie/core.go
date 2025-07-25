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

	cfg.Routing = func(m django.Mux) {
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

		return nil
	}

	cfg.Ready = func() error {
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
					admin.MultiPanel(
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
					admin.MultiPanel(
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
