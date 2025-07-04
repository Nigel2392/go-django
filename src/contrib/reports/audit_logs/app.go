package auditlogs

import (
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"unicode"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/contrib/filters"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/pkg/errors"

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
			if err := schemaEditor.CreateTable(table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		goldcrest.Register(
			reports.ReportsMenuHook, 0,
			reports.ReportsMenuHookFunc(func() []menu.MenuItem {
				var auditLogItem = &menu.Item{
					BaseItem: menu.BaseItem{
						Label: trans.S("Audit Logs"),
					},
					Link: func() string {
						return django.Reverse("admin:auditlogs")
					},
				}

				return []menu.MenuItem{auditLogItem}
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
		return m
	})

	var hookNames = []string{
		admin.AdminModelHookAdd,
		admin.AdminModelHookEdit,
		admin.AdminModelHookDelete,
	}

	for _, hookName := range hookNames {
		var hookName = hookName
		goldcrest.Register(hookName, 0, admin.AdminModelHookFunc(
			func(r *http.Request, adminSite *admin.AdminApplication, model *admin.ModelDefinition, instance attrs.Definer) {
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
			}),
		)
	}

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

func isNumber(v string) bool {
	for _, c := range v {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
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
		"filters",
	)

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName:  "type",
		FormField: fields.CharField(),
		Apply: func(value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("Type", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName:  "object_id",
		FormField: fields.CharField(),
		Apply: func(value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("ObjectID", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName:  "content_type",
		FormField: fields.CharField(),
		Apply: func(value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			return object.Filter("ContentType", value), nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[*queries.QuerySet[*Entry]]{
		SpecName: "level",
		FormField: fields.CharField(fields.Widget(
			options.NewSelectInput(nil, func() []widgets.Option {
				return []widgets.Option{
					&widgets.FormOption{OptValue: strconv.Itoa(int(logger.DBG)), OptLabel: trans.T("Debug")},
					&widgets.FormOption{OptValue: strconv.Itoa(int(logger.INF)), OptLabel: trans.T("Info")},
					&widgets.FormOption{OptValue: strconv.Itoa(int(logger.WRN)), OptLabel: trans.T("Warning")},
					&widgets.FormOption{OptValue: strconv.Itoa(int(logger.ERR)), OptLabel: trans.T("Error")},
				}
			}),
		)),
		Apply: func(value interface{}, object *queries.QuerySet[*Entry]) (*queries.QuerySet[*Entry], error) {
			if fields.IsZero(value) {
				return object, nil
			}

			var v = value.(string)
			var level, err = strconv.Atoi(v)
			if err != nil {
				return nil, errors.Wrapf(err, "Invalid log level: %s", v)
			}
			return object.Filter("Level", logger.LogLevel(level)), nil
		},
	})

	var (
		err error
		qs  = queries.GetQuerySet(&Entry{})
	)

	qs, err = filter.Filter(r.URL.Query(), qs)
	if err != nil {
		logger.Errorf("Failed to filter audit logs: %v", err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to filter audit logs",
		)
		return
	}

	var paginator = pagination.Paginator[[]LogEntry, LogEntry]{
		GetObject: func(l LogEntry) LogEntry {
			return Define(r, l)
		},
		GetObjects: func(i1, i2 int) ([]LogEntry, error) {
			objectRows, err := qs.
				Offset(i1).
				Limit(i2).
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
		Amount: 15,
	}

	var pageNum = pagination.GetPageNum(
		r.URL.Query().Get("page"),
	)

	if pageNum < 1 {
		pageNum = 1
	}

	page, err := paginator.Page(pageNum)
	if err != nil && !errors.Is(err, pagination.ErrNoResults) {
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

	adminCtx.Set("content_type", filter.Form().CleanedData()["content_type"])
	adminCtx.Set("object_id", filter.Form().CleanedData()["object_id"])
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
