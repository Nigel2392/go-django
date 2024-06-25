package auditlogs

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/admin/components"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/core/pagination"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"

	"embed"
)

type AuditLogs struct {
	*apps.AppConfig
}

var Logs *AuditLogs = &AuditLogs{
	AppConfig: apps.NewAppConfig("auditlogs"),
}

//go:embed assets/*
var templateFileSys embed.FS

func NewAppConfig() django.AppConfig {
	Logs.Init = func(settings django.Settings) error {

		if registry.backend != nil {
			var err = registry.backend.Setup()
			if err != nil {
				return err
			}
		}

		//dbInt, ok := settings.Get("DATABASE")
		//assert.True(ok, "DATABASE setting is required for 'auditlogs' app")
		//
		//_, ok = dbInt.(*sql.DB)
		//assert.True(ok, "DATABASE setting must adhere to auditlogs-models.DBTX interface")
		//
		// Logs.Queries = models.NewQueries(db)
		fmt.Println("Initializing audit logs app")
		goldcrest.Register(
			admin.RegisterMenuItemHook, 0,
			admin.RegisterMenuItemHookFunc(func(adminSite *admin.AdminApplication, items components.Items[menu.MenuItem]) {
				//var auditLogItem = menu.SubmenuItem{
				//	BaseItem: menu.BaseItem{
				//		Label: fields.S("Audit Logs"),
				//	},
				//	Menu: &menu.Menu{
				//		Items: []menu.MenuItem{
				//			&menu.BaseItem{
				//				Label: fields.S("View Logs"),
				//			},
				//			&menu.SubmenuItem{
				//				BaseItem: menu.BaseItem{
				//					Label: fields.S("Audit Logs"),
				//				},
				//				Menu: &menu.Menu{
				//					Items: []menu.MenuItem{
				//						&menu.BaseItem{
				//							Label: fields.S("View Logs"),
				//						},
				//					},
				//				},
				//			},
				//			&menu.SubmenuItem{
				//				BaseItem: menu.BaseItem{
				//					Label: fields.S("Audit Logs"),
				//				},
				//				Menu: &menu.Menu{
				//					Items: []menu.MenuItem{
				//						&menu.BaseItem{
				//							Label: fields.S("View Logs"),
				//						},
				//						&menu.SubmenuItem{
				//							BaseItem: menu.BaseItem{
				//								Label: fields.S("Audit Logs"),
				//							},
				//							Menu: &menu.Menu{
				//								Items: []menu.MenuItem{
				//									&menu.BaseItem{
				//										Label: fields.S("View Logs"),
				//									},
				//								},
				//							},
				//						},
				//					},
				//				},
				//			},
				//		},
				//	},
				//}
				var auditLogItem = &menu.Item{
					BaseItem: menu.BaseItem{
						Label: fields.S("Audit Logs"),
					},
					Link: func() string {
						return django.Reverse("admin:auditlogs")
					},
				}

				items.Append(auditLogItem)
			}),
		)

		return nil
	}

	var tplFS, err = fs.Sub(templateFileSys, "assets/templates")
	if err != nil {
		panic(err)
	}

	sFs, err := fs.Sub(templateFileSys, "assets/static")
	if err != nil {
		panic(err)
	}

	staticfiles.AddFS(
		sFs, tpl.MatchAnd(
			tpl.MatchPrefix("auditlogs/"),
			tpl.MatchOr(
				tpl.MatchSuffix(".css"),
				tpl.MatchSuffix(".js"),
			),
		),
	)

	admin.RegisterMedia(func(adminSite *admin.AdminApplication) media.Media {
		var m = media.NewMedia()
		m.AddCSS(media.CSS(django.Static("auditlogs/css/auditlogs.css")))
		return m
	})

	tpl.Add(tpl.Config{
		AppName: "auditlogs",
		FS:      tplFS,
		Matches: tpl.MatchAnd(
			tpl.MatchPrefix("auditlogs/"),
			tpl.MatchOr(
				tpl.MatchSuffix(".tmpl"),
			),
		),
	})

	Logs.Ready = func() error {
		admin.AdminSite.Route.Handle(
			mux.ANY, "/auditlogs", mux.NewHandler(auditLogView),
			"auditlogs",
		)
		return nil
	}

	return Logs
}

func auditLogView(w http.ResponseWriter, r *http.Request) {

	var adminCtx = admin.NewContext(r, admin.AdminSite, nil)

	var backend = Backend()
	if backend == nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to setup audit logs backend",
		)
		return
	}

	var paginator = pagination.Paginator[LogEntry]{
		GetObject: func(l LogEntry) LogEntry {
			return Define(r, l)
		},
		GetObjects: backend.RetrieveMany,
		GetCount:   backend.Count,
		Amount:     15,
	}

	var pageNum = pagination.GetPageNum(
		r.URL.Query().Get("page"),
	)

	var page, err = paginator.Page(pageNum)
	if err != nil {
		logger.Error(err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to retrieve audit logs",
		)
		return
	}

	//var definitions = make([]*BoundDefinition, page.Count())
	//for i, log := range page.Results() {
	//	definitions[i] = Define(r, log)
	//}

	adminCtx.Set("paginator", page)
	adminCtx.SetPage(admin.PageOptions{
		TitleFn:    fields.S("Audit Logs"),
		SubtitleFn: fields.S("View all audit logs"),
	})

	var v = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: "admin",
		TemplateName:    "auditlogs/views/logs.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			return adminCtx, nil
		},
	}

	if err = views.Invoke(v, w, r); err != nil {
		return
	}
}
