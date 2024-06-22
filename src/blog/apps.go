package blog

import (
	"context"
	"net/http"

	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/pages"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/forms/fields"
)

var blog *apps.DBRequiredAppConfig

func NewAppConfig() *apps.DBRequiredAppConfig {
	var appconfig = apps.NewDBAppConfig("blog")
	appconfig.Ready = func() error {
		pages.Register(&pages.PageDefinition{
			ContentTypeDefinition: &contenttypes.ContentTypeDefinition{
				GetLabel:       fields.S("Blog Page"),
				GetDescription: fields.S("A blog page with a rich text editor."),
				ContentObject:  &BlogPage{},
				Aliases: []string{
					"github.com/Nigel2392/src/core.BlogPage",
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
			GetForID: func(ctx context.Context, ref models.PageNode, id int64) (pages.Page, error) {
				return getBlogPage(ref, id)
			},
		})
		blog = appconfig
		return nil
	}
	return appconfig
}
