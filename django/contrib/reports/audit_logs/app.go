package auditlogs

import (
	"io/fs"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/views"
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
			return registry.backend.Setup()
		}

		//dbInt, ok := settings.Get("DATABASE")
		//assert.True(ok, "DATABASE setting is required for 'auditlogs' app")
		//
		//_, ok = dbInt.(*sql.DB)
		//assert.True(ok, "DATABASE setting must adhere to auditlogs-models.DBTX interface")
		//
		//// Logs.Queries = models.NewQueries(db)
		//goldcrest.Register(
		//	admin.RegisterMenuItemHook, 100,
		//	admin.RegisterMenuItemHookFunc(func(adminSite *admin.AdminApplication, items menu.Items) {
		//
		//		var auditLogItem = menu.SubmenuItem{
		//			BaseItem: menu.BaseItem{
		//				Label: fields.S("Audit Logs"),
		//			},
		//			Menu: &menu.Menu{
		//				Items: []menu.MenuItem{
		//					&menu.BaseItem{
		//						Label: fields.S("View Logs"),
		//					},
		//					&menu.SubmenuItem{
		//						BaseItem: menu.BaseItem{
		//							Label: fields.S("Audit Logs"),
		//						},
		//						Menu: &menu.Menu{
		//							Items: []menu.MenuItem{
		//								&menu.BaseItem{
		//									Label: fields.S("View Logs"),
		//								},
		//							},
		//						},
		//					},
		//					&menu.SubmenuItem{
		//						BaseItem: menu.BaseItem{
		//							Label: fields.S("Audit Logs"),
		//						},
		//						Menu: &menu.Menu{
		//							Items: []menu.MenuItem{
		//								&menu.BaseItem{
		//									Label: fields.S("View Logs"),
		//								},
		//								&menu.SubmenuItem{
		//									BaseItem: menu.BaseItem{
		//										Label: fields.S("Audit Logs"),
		//									},
		//									Menu: &menu.Menu{
		//										Items: []menu.MenuItem{
		//											&menu.BaseItem{
		//												Label: fields.S("View Logs"),
		//											},
		//										},
		//									},
		//								},
		//							},
		//						},
		//					},
		//				},
		//			},
		//		}
		//
		//		items.Append(&auditLogItem)
		//	}),
		//)

		return nil
	}

	var tplFS, err = fs.Sub(templateFileSys, "assets/templates")
	if err != nil {
		panic(err)
	}

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

	var logList, err = Backend().RetrieveMany(25, 0)
	if err != nil {

		logger.Error(err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to retrieve audit logs",
		)
		return
	}

	var definitions = make([]*BoundDefinition, len(logList))
	for i, log := range logList {
		definitions[i] = Define(log)
	}

	adminCtx.Set("logs", definitions)

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
