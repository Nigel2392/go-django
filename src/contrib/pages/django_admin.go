package pages

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/pkg/errors"
)

const (
	AdminPagesAppName   = "pages"
	AdminPagesModelPath = "Page"
)

var pageAdminAppOptions = admin.AppOptions{
	AppLabel:       trans.S("Pages"),
	AppDescription: trans.S("Manage pages in a hierarchical structure."),
	MediaFn: func() media.Media {
		var m = media.NewMedia()
		m.AddCSS(
			media.CSS(django.Static("pages/admin/css/index.css")),
		)
		m.AddJS(
			media.JS(django.Static("pages/admin/js/index.js")),
		)
		return m
	},
}

type TimeInformation struct {
	Source  time.Time
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
	Future  bool
}

func GetTimeDiffInformation(t time.Time, diffFrom time.Time) (info TimeInformation) {
	other := diffFrom.In(t.Location())
	if t.After(other) {
		t, other = other, t // swap, so we always diff "earlier to later"
		info.Future = true
	}

	temp := t

	// Years
	for temp.AddDate(1, 0, 0).Before(other) || temp.AddDate(1, 0, 0).Equal(other) {
		temp = temp.AddDate(1, 0, 0)
		info.Years++
	}

	// Months
	for temp.AddDate(0, 1, 0).Before(other) || temp.AddDate(0, 1, 0).Equal(other) {
		temp = temp.AddDate(0, 1, 0)
		info.Months++
	}

	// Days
	for temp.AddDate(0, 0, 1).Before(other) || temp.AddDate(0, 0, 1).Equal(other) {
		temp = temp.AddDate(0, 0, 1)
		info.Days++
	}

	duration := other.Sub(temp)
	info.Source = t
	info.Hours = int(duration.Hours())
	info.Minutes = int(duration.Minutes()) % 60
	info.Seconds = int(duration.Seconds()) % 60
	return info
}

// FormatTimeAgo formats the time difference nicely, e.g. "1 year, 2 months, 3 weeks ago"
func FormatTimeDifference(ctx context.Context, t time.Time, diffFrom time.Time) (string, TimeInformation) {
	var parts = []string{}
	var info = GetTimeDiffInformation(t, diffFrom)
	if info.Years > 0 {
		parts = append(parts, trans.P(ctx, "%d year", "%d years", info.Years, info.Years))
	}
	if info.Months > 0 {
		parts = append(parts, trans.P(ctx, "%d month", "%d months", info.Months, info.Months))
	}
	if info.Days >= 7 {
		weeks := info.Days / 7
		parts = append(parts, trans.P(ctx, "%d week", "%d weeks", weeks, weeks))
		info.Days = info.Days % 7
	}
	if info.Days > 0 && len(parts) < 2 {
		parts = append(parts, trans.P(ctx, "%d day", "%d days", info.Days, info.Days))
	}
	if info.Hours > 0 && len(parts) == 0 {
		parts = append(parts, trans.P(ctx, "%d hour", "%d hours", info.Hours, info.Hours))
	}
	if info.Minutes > 0 && len(parts) == 0 {
		parts = append(parts, trans.P(ctx, "%d minute", "%d minutes", info.Minutes, info.Minutes))
	}

	// Only show up to two largest units, e.g. "1 year, 2 months ago"
	if len(parts) > 2 {
		parts = parts[:2]
	}

	var text string
	if info.Future {
		switch len(parts) {
		case 0:
		case 1:
			text = trans.T(ctx, "in %s", parts[0])
		default:
			text = trans.T(ctx, "in %s and %s", parts[0], parts[1])
		}
	} else {
		switch len(parts) {
		case 0:
			text = trans.T(ctx, "just now")
		case 1:
			text = trans.T(ctx, "%s ago", parts[0])
		default:
			text = trans.T(ctx, "%s and %s ago", parts[0], parts[1])
		}
	}

	return text, info
}

var pageAdminModelOptions = admin.ModelOptions{
	Name:  AdminPagesModelPath,
	Model: &PageNode{},
	Labels: map[string]func(ctx context.Context) string{
		"ID":          trans.S("ID"),
		"Title":       trans.S("Title"),
		"Path":        trans.S("Tree Path"),
		"Depth":       trans.S("Tree Depth"),
		"Numchild":    trans.S("Number of Children"),
		"UrlPath":     trans.S("URL Path"),
		"Slug":        trans.S("Slug"),
		"StatusFlags": trans.S("Status Flags"),
		"PageID":      trans.S("Page ID"),
		"ContentType": trans.S("Content Type"),
		"CreatedAt":   trans.S("Created At"),
		"UpdatedAt":   trans.S("Updated At"),
	},
	ListView: admin.ListViewOptions{
		ViewOptions: admin.ViewOptions{
			Fields: []string{
				"Title",
				"Slug",
				"ContentType",
				"CreatedAt",
				"UpdatedAt",
				"Children",
			},
		},
		Columns: map[string]list.ListColumn[attrs.Definer]{
			"Children": list.HTMLColumn[attrs.Definer](
				func(ctx context.Context) string { return "" },
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer) template.HTML {
					var node = row.(*PageNode)
					if node.Numchild > 0 {
						var url = django.Reverse(
							"admin:pages:list",
							node.PK,
						)

						return template.HTML(fmt.Sprintf(
							`<a href="%s" class="button primary hollow">%s</a>`,
							url, trans.P(
								r.Context(), "%d child", "%d children",
								int(node.Numchild), node.Numchild,
							),
						))
					}
					return template.HTML("")
				},
			),
			"Live": list.BooleanColumn(
				trans.S("Live"),
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer) bool {
					var node = row.(*PageNode)
					return node.IsPublished()
				},
			),
			"UpdatedAt": list.HTMLColumn(
				trans.S("Time since last update"),
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer) template.HTML {
					var node = row.(*PageNode)

					if node.UpdatedAt.IsZero() {
						return template.HTML(fmt.Sprintf(
							`<span class="badge warning">%s</span>`,
							trans.T(r.Context(), "Never"),
						))
					}

					var timeText, _ = FormatTimeDifference(r.Context(), node.UpdatedAt, time.Now())
					return template.HTML(fmt.Sprintf(
						`<span class="badge" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="%s" data-tooltip-offset-value="[0, %v]">%s</span>`,
						trans.Time(r.Context(), node.UpdatedAt, trans.LONG_TIME_FORMAT), "bottom-start", 10, timeText,
					))
				},
			),
			"ContentType": list.HTMLColumn(
				trans.S("Content Type"),
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer) template.HTML {
					var node = row.(*PageNode)
					var ctype = DefinitionForType(node.ContentType)
					var typStr = trans.T(r.Context(), "Unknown")
					if ctype != nil {
						typStr = ctype.Label(r.Context())
					}

					return template.HTML(fmt.Sprintf(
						`<span class="badge">%s</span>`,
						typStr,
					))
				},
			),
			"UrlPath": list.HTMLColumn(
				trans.S("URL Path"),
				func(r *http.Request, _ attrs.Definitions, row attrs.Definer) template.HTML {
					var node = row.(*PageNode)

					if !node.IsPublished() {
						return template.HTML(fmt.Sprintf(
							`<span class="badge warning">%s</span>`,
							node.UrlPath,
						))
					}

					return template.HTML(fmt.Sprintf(
						`<a href="%s">%s</a>`,
						URLPath(node), node.UrlPath,
					))
				},
			),
		},
		PerPage: 16,
	},
	AddView: admin.FormViewOptions{
		SaveInstance: saveInstanceFunc,
		Panels: []admin.Panel{
			admin.TitlePanel(
				admin.FieldPanel("Title"),
			),
			admin.MultiPanel(
				admin.FieldPanel("Path"),
				admin.FieldPanel("Depth"),
			),
			admin.FieldPanel("PageID"),
			admin.FieldPanel("Numchild"),
			admin.FieldPanel("UrlPath"),
			admin.FieldPanel("Slug"),
			admin.FieldPanel("StatusFlags"),
		},
		// Change form interaction
	},
	EditView: admin.FormViewOptions{
		SaveInstance: saveInstanceFunc,
		// Change form interaction
		Panels: []admin.Panel{
			admin.TitlePanel(
				admin.FieldPanel("Title"),
			),
			admin.MultiPanel(
				admin.FieldPanel("Path"),
				admin.FieldPanel("Depth"),
			),
			admin.FieldPanel("PageID"),
			admin.FieldPanel("Numchild"),
			admin.FieldPanel("UrlPath"),
			admin.FieldPanel("Slug"),
			admin.FieldPanel("StatusFlags"),
			admin.FieldPanel("CreatedAt"),
			admin.FieldPanel("UpdatedAt"),
		},
	},
}

func saveInstanceFunc(ctx context.Context, d attrs.Definer) error {
	var n = d.(*PageNode)
	var err error
	var qs = NewPageQuerySet().WithContext(ctx)
	if n.PK == 0 {
		_, err = qs.ExplicitSave().Create(n)
	} else {
		err = qs.UpdateNode(n)
	}

	if err != nil {
		return err
	}

	//django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
	//	return FixTree(pageApp.QuerySet(), ctx)
	//})

	return nil
}

// Fixtree fixes the tree structure of the page nodes.
//
// It scans for errors in the tree structure in the database and fixes them.
func FixTree(ctx context.Context) error {
	var querySet = queries.GetQuerySet(&PageNode{}).WithContext(ctx)
	var transaction, err = querySet.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(ctx)

	var qs = NewPageQuerySet().WithContext(ctx)
	allNodesCount, err := qs.Count()
	if err != nil {
		return errors.Wrap(err, "failed to count nodes")
	}

	allNodes, err := qs.Offset(0).Limit(int(allNodesCount)).AllNodes()
	if err != nil {
		return errors.Wrap(err, "failed to get all nodes")
	}

	var tree = NewNodeTree(allNodes)

	tree.FixTree()

	err = qs.updateNodes(allNodes)
	if err != nil {
		return errors.Wrap(err, "failed to update nodes")
	}

	return transaction.Commit(ctx)
}
