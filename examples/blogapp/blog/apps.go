package blog

import (
	"embed"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/pkg/errors"
)

type customCommandObj struct {
	printTime bool
	printText string
}

var myCustomCommand = &command.Cmd[customCommandObj]{
	ID:   "mycommand",
	Desc: "Prints the provided text with an optional timestamp",

	FlagFunc: func(m command.Manager, stored *customCommandObj, f *flag.FlagSet) error {
		f.BoolVar(&stored.printTime, "t", false, "Print the current time")
		f.StringVar(&stored.printText, "text", "", "The text to print")
		return nil
	},

	Execute: func(m command.Manager, stored customCommandObj, args []string) error {
		if stored.printText == "" {
			return errors.New("No text provided")
		}

		if stored.printTime {
			fmt.Println(time.Now().Format(time.RFC3339), stored.printText)
		} else {
			fmt.Println(stored.printText)
		}
		return nil
	},
}

var blog *apps.DBRequiredAppConfig

//go:embed assets/**
var blogFS embed.FS

func NewAppConfig() *apps.DBRequiredAppConfig {
	var appconfig = apps.NewDBAppConfig("blog")

	appconfig.ModelObjects = []attrs.Definer{
		&BlogPage{},
	}

	appconfig.Init = func(settings django.Settings, db drivers.Database) error {
		var (
			tplFS    = filesystem.Sub(blogFS, "assets/templates")
			staticFS = filesystem.Sub(blogFS, "assets/static")
		)

		// Set up the static files for this app
		// They are stored in the "assets/static" directory
		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchPrefix("blog/"),
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

		// Set up the templates for this app
		// They are stored in the "assets/templates" directory
		tpl.Add(tpl.Config{
			AppName: "blog",
			FS:      tplFS,
			Bases: []string{
				"blog/base.tmpl",
			},
			Matches: filesystem.MatchAnd(
				filesystem.MatchPrefix("blog/"),
				filesystem.MatchExt(".tmpl"),
				filesystem.MatchExt(".tmpl"),
			),
		})

		return nil
	}

	appconfig.Ready = func() error {
		pages.Register(&pages.PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				GetLabel:       trans.S("Blog Page"),
				GetDescription: trans.S("A blog page with a rich text editor."),
				ContentObject:  &BlogPage{},
				Aliases: []string{
					"github.com/Nigel2392/go-django/example/blogapp/core.BlogPage",
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
					admin.FieldPanel("Editor"),
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
					admin.FieldPanel("Editor"),
					admin.FieldPanel("CreatedAt"),
					admin.FieldPanel("UpdatedAt"),
				}
			},
			ParentPageTypes: []string{
				"github.com/Nigel2392/go-django-example/src/blog.BlogPage",
			},
			//GetForID: func(ctx context.Context, ref *pages.PageNode, id int64) (pages.Page, error) {
			//	var row, err = queries.GetQuerySet(&BlogPage{}).Filter("PageID", id).First()
			//	if err != nil {
			//		return nil, errors.Wrapf(err, "failed to get blog page with ID %d", id)
			//	}
			//	*row.Object.PageNode = *ref
			//	return row.Object, nil
			//},
		})
		blog = appconfig
		return nil
	}

	return appconfig
}
