package blog

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
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

func NewAppConfig() *apps.DBRequiredAppConfig {
	var appconfig = apps.NewDBAppConfig("blog")

	appconfig.ModelObjects = []attrs.Definer{
		&BlogPage{},
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
