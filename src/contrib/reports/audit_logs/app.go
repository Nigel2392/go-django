package auditlogs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/filters"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/secrets/safety"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/a-h/templ"

	"embed"
)

type AuditLogs struct {
	*apps.DBRequiredAppConfig
}

var Logs *AuditLogs = &AuditLogs{
	DBRequiredAppConfig: apps.NewDBAppConfig("auditlogs"),
}

//go:embed assets/*
var templateFileSys embed.FS

//go:embed migrations/*
var migrationFileSys embed.FS

func NewAppConfig() django.AppConfig {
	Logs.Deps = []string{
		"reports",
	}

	Logs.ModelObjects = []attrs.Definer{
		&Entry{},
	}

	Logs.Init = func(settings django.Settings, db drivers.Database) error {

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&Entry{})
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(context.Background(), table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		reports.RegisterMenuItem(func(r *http.Request) []menu.MenuItem {
			var auditLogItem = &menu.Item{
				BaseItem: menu.BaseItem{
					Label: trans.T(r.Context(), "Audit Logs"),
				},
				Link: func() string {
					return django.Reverse("admin:auditlogs")
				},
			}

			return []menu.MenuItem{auditLogItem}
		})

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
		sFs, filesystem.MatchAnd(
			filesystem.MatchPrefix("auditlogs/"),
			filesystem.MatchOr(
				filesystem.MatchSuffix(".css"),
				filesystem.MatchSuffix(".js"),
			),
		),
	)

	admin.RegisterGlobalMedia(func(adminSite *admin.AdminApplication) media.Media {
		var m = media.NewMedia()
		m.AddCSS(media.CSS(django.Static("auditlogs/css/auditlogs.css")))
		m.AddJS(media.JS(django.Static("auditlogs/js/auditlogs.js")))
		return m
	})

	var hookNames = []string{
		admin.AdminModelHookAdd,
		admin.AdminModelHookEdit,
	}

	var hookFunc = func(hookName string, r *http.Request, _ *admin.AdminApplication, model *admin.ModelDefinition, instance attrs.Definer) {
		if !Logs.IsReady() {
			return
		}

		var data = make(map[string]interface{})
		var level logger.LogLevel = logger.DBG
		switch hookName {
		case admin.AdminModelHookAdd:
			level = logger.INF
		case admin.AdminModelHookEdit:
			level = logger.INF
		case admin.AdminModelHookDelete:
			level = logger.WRN
		}

		data["model_name"] = model.Name
		data["instance_id"] = attrs.PrimaryKey(instance)
		var cTypeDef = contenttypes.DefinitionForObject(instance)
		if cTypeDef != nil {
			data["content_type"] = cTypeDef.ContentType()
		}

		if _, err := Log(r.Context(), hookName, level, instance, data); err != nil {
			logger.Warn(err)
		}
	}

	for _, hookName := range hookNames {
		var hookName = hookName
		goldcrest.Register(hookName, 0, admin.AdminModelHookFunc(func(r *http.Request, adminSite *admin.AdminApplication, model *admin.ModelDefinition, instance attrs.Definer) {
			hookFunc(hookName, r, adminSite, model, instance)
		}))
	}

	core.SIGNAL_LOGIN_FAILED.Listen(func(s signals.Signal[*http.Request], r *http.Request) error {
		var logData = make(map[string]interface{})
		var newData = make(url.Values, len(r.Form))
		for k, v := range r.Form {
			if safety.IsSecretField(r.Context(), k) {
				newData[k] = []string{"**********"}
			} else {
				newData[k] = slices.Clone(v)
			}
		}

		logData["form"] = newData
		logData["host"] = mux.GetHost(r)
		logData["ip"] = django.GetIP(r)

		if _, err := Log(r.Context(), "auth.login_failed", logger.ERR, nil, logData); err != nil {
			logger.Warn(err)
		}

		return nil
	})

	goldcrest.Register(admin.AdminModelHookDelete, 0, admin.AdminModelDeleteFunc(func(r *http.Request, adminSite *admin.AdminApplication, model *admin.ModelDefinition, instances []attrs.Definer) {
		for _, instance := range instances {
			hookFunc(admin.AdminModelHookDelete, r, adminSite, model, instance)
		}
	}))

	RegisterDefinition("auth.login_failed", &loginFailedDefinition{})

	if admin.AdminSite.TemplateConfig != nil {
		Logs.TemplateConfig = &tpl.Config{
			AppName: "auditlogs",
			FS: filesystem.NewMultiFS(
				filesystem.NewMatchFS(
					tplFS,
					filesystem.MatchAnd(
						filesystem.MatchPrefix("auditlogs/"),
						filesystem.MatchOr(
							filesystem.MatchSuffix(".tmpl"),
						),
					),
				),
				filesystem.NewMatchFS(
					admin.AdminSite.TemplateConfig.FS,
					admin.AdminSite.TemplateConfig.Matches,
				),
			),
			Bases: admin.AdminSite.TemplateConfig.Bases,
			Funcs: admin.AdminSite.TemplateConfig.Funcs,
		}
	}

	Logs.Ready = func() error {
		admin.AdminSite.Route.Handle(
			mux.ANY, "/auditlogs", mux.NewHandler(auditLogView),
			"auditlogs",
		)
		return nil
	}

	return &migrator.MigratorAppConfig{
		AppConfig: Logs,
		MigrationFS: filesystem.Sub(
			migrationFileSys, "migrations/auditlogs",
		),
	}
}

func auditLogView(w http.ResponseWriter, r *http.Request) {

	if !permissions.HasPermission(r, "auditlogs:list") {
		except.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	var adminCtx = admin.NewContext(r, admin.AdminSite, nil)

	var filter = filters.NewFilters[*Entry](
		r.Context(), "filters",
	)

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName: "type",
		FormField: fields.CharField(fields.Widget(
			options.NewSelectInput(nil, func() []widgets.Option {
				var vals, err = queries.
					GetQuerySet(&Entry{}).
					Filter("Type__isnull", false).
					Distinct().
					OrderBy("Type").
					ValuesList("Type")
				if err != nil {
					logger.Errorf("Failed to get types for audit logs: %v", err)
					except.Fail(
						http.StatusInternalServerError,
						"Failed to get types for audit logs",
					)
					return nil
				}

				var opts = make([]widgets.Option, len(vals))
				for i, val := range vals {
					if len(val) == 0 {
						continue
					}

					var v = reflect.ValueOf(val[0]).String()
					var def = DefinitionForType(v)
					if def == nil {
						def = SimpleDefinition()
					}

					opts[i] = &widgets.FormOption{
						OptValue: v,
						OptLabel: def.TypeLabel(r, v),
					}
				}

				return opts
			}, options.IncludeBlank(true)),
		)),
		Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("Type", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName:  "object_id",
		FormField: fields.CharField(),
		Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("ObjectID", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName: "content_type",
		FormField: fields.CharField(fields.Widget(
			options.NewSelectInput(nil, func() []widgets.Option {
				var vals, err = queries.
					GetQuerySet(&Entry{}).
					Distinct().
					Filter(expr.Q("ContentType", "").Not(true)).
					ValuesList("ContentType")
				if err != nil {
					logger.Errorf("Failed to get content types for audit logs: %v", err)
					except.Fail(
						http.StatusInternalServerError,
						"Failed to get content types for audit logs",
					)
					return nil
				}

				var opts = make([]widgets.Option, len(vals))
				for i, val := range vals {
					var cType = val[0].(*contenttypes.BaseContentType[interface{}])
					opts[i] = &widgets.FormOption{
						OptValue: cType.ShortTypeName(),
						OptLabel: trans.T(r.Context(), cType.Model()),
					}
				}

				return opts
			}, options.IncludeBlank(true)),
		)),
		Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("ContentType__endswith", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName: "user",
		FormField: fields.CharField(fields.Widget(
			options.NewSelectInput(nil, func() []widgets.Option {
				var userModelDef = users.GetUserModel()
				var userModel = userModelDef.
					ContentType().
					New().(users.User)

				var meta = attrs.GetModelMeta(userModel)
				var defs = meta.Definitions()
				var pkField = defs.Primary()

				var rowCnt, rowIter, err = queries.
					GetQuerySet(userModel).
					Filter(
						fmt.Sprintf("%s__in", pkField.Name()),
						queries.Objects(&Entry{}).
							Select("User").
							Distinct(),
					).
					Distinct().
					IterAll()
				if err != nil {
					logger.Errorf("Failed to get users for audit logs: %v", err)
					except.Fail(
						http.StatusInternalServerError,
						"Failed to get users for audit logs",
					)
					return nil
				}

				var idx = 0
				var opts = make([]widgets.Option, rowCnt)
				for row, err := range rowIter {
					if err != nil {
						logger.Errorf("Failed to iterate users for audit logs: %v", err)
						except.Fail(
							http.StatusInternalServerError,
							"Failed to iterate users for audit logs",
						)
						return nil
					}

					opts[idx] = &widgets.FormOption{
						OptValue: attrs.ToString(attrs.PrimaryKey(row.Object)),
						OptLabel: attrs.ToString(row.Object),
					}

					idx++
				}

				return opts
			}, options.IncludeBlank(true)),
		)),
		Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("User", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName: "level",
		FormField: fields.CharField(fields.Widget(
			options.NewSelectInput(nil, func() []widgets.Option {
				return []widgets.Option{
					&widgets.FormOption{OptValue: logger.DBG.String(), OptLabel: trans.T(r.Context(), "Debug")},
					&widgets.FormOption{OptValue: logger.INF.String(), OptLabel: trans.T(r.Context(), "Info")},
					&widgets.FormOption{OptValue: logger.WRN.String(), OptLabel: trans.T(r.Context(), "Warning")},
					&widgets.FormOption{OptValue: logger.ERR.String(), OptLabel: trans.T(r.Context(), "Error")},
				}
			}, options.IncludeBlank(true)),
		)),
		Apply: func(req *http.Request, value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			var level, err = logger.ParseLogLevel(value.(string))
			if err != nil {
				return nil, errors.Wrapf(err, "Invalid log level: %v", value)
			}
			return object.Filter("Level", logger.LogLevel(level)), nil
		},
	})

	filter.Form().Ordering([]string{
		"type",
		"level",
		"user",
		"content_type",
		"object_id",
	})

	var err error
	var qs = queries.
		GetQuerySet(&Entry{}).
		SelectRelated("User").
		OrderBy("-Timestamp")

	qs, err = filter.Filter(r, r.URL.Query(), qs)
	if err != nil && !errors.Is(err, filters.ErrForm) {
		logger.Errorf("Failed to filter audit logs: %v", err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to filter audit logs",
		)
		return
	}

	var paginator = pagination.Paginator[[]LogEntry, LogEntry]{
		Context: r.Context(),
		GetObject: func(l LogEntry) LogEntry {
			return Define(r, l)
		},
		GetObjects: func(i1, i2 int) ([]LogEntry, error) {
			objectRows, err := qs.
				Offset(i2).
				Limit(i1).
				All()

			if err != nil {
				return nil, err
			}

			var objects = make([]LogEntry, len(objectRows))
			for i, row := range objectRows {
				objects[i] = row.Object
			}

			return objects, nil
		},
		GetCount: func() (int, error) {
			var count, err = qs.Count()
			if err != nil {
				return 0, errors.Wrap(err, "Failed to count audit logs")
			}
			return int(count), nil
		},
		Amount: 25,
	}

	var amount, _ = strconv.Atoi(r.URL.Query().Get("amount"))
	if amount < 1 {
		amount = 25
	}

	paginator.Amount = amount

	var pageNum = pagination.GetPageNum(
		r.URL.Query().Get("page"),
	)

	if pageNum < 1 {
		pageNum = 1
	}

	page, err := paginator.Page(pageNum)
	if err != nil && !errors.Is(err, errors.NoRows) {
		logger.Errorf("Failed to retrieve audit logs: %v", err)
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
	adminCtx.Set("form", filter.Form())
	adminCtx.Set(
		"actionURL",
		django.Reverse(
			"admin:auditlogs",
		),
	)

	adminCtx.SetPage(admin.PageOptions{
		TitleFn:    trans.S("Audit Logs"),
		SubtitleFn: trans.S("View all audit logs"),
		SidePanels: &menu.SidePanels{
			ActivePanel: "filters",
			Panels: []menu.SidePanel{
				admin.SidePanelFilters(r, filter, page),
			},
		},
		Buttons: []components.ShowableComponent{
			components.NewShowableComponent(r, nil, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				var q = maps.Clone(r.URL.Query())
				q.Del("page")
				q.Del("amount")

				w.Write([]byte(`<form method="GET" action="`))
				w.Write([]byte(django.Reverse("admin:auditlogs")))
				w.Write([]byte("?"))
				w.Write([]byte(q.Encode()))
				w.Write([]byte(`" class="auditlogs-amount-form form-field">`))

				for k, v := range q {
					if len(v) > 0 {
						w.Write([]byte(`<input type="hidden" name="`))
						w.Write([]byte(k))
						w.Write([]byte(`" value="`))
						w.Write([]byte(v[0]))
						w.Write([]byte(`">`))
					}
				}

				w.Write([]byte(`<select name="amount" onchange="this.form.submit()">`))
				for _, v := range []int{15, 25, 50, 100} {
					var selectedText string
					if v == amount {
						selectedText = ` selected`
					}

					fmt.Fprintf(w, `<option value="%d"%s>%d</option>`, v, selectedText, v)
				}
				w.Write([]byte(`</select>`))
				w.Write([]byte(`</form>`))
				return nil
			})),
		},
	})

	if err := tpl.FRender(w, adminCtx, "admin", "auditlogs/views/logs.tmpl"); err != nil {
		logger.Errorf("Failed to render audit logs template: %v", err)
		return
	}
}
