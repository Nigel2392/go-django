package auditlogs

import (
	"io/fs"
	"net/http"
	"strconv"
	"unicode"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/contrib/filters"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	"github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/backend"
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
	"github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/pkg/errors"

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
	Logs.Deps = []string{
		"reports",
	}
	Logs.Init = func(settings django.Settings) error {

		if registry.backend == nil {
			var db = django.ConfigGet[drivers.Database](
				django.Global.Settings,
				django.APPVAR_DATABASE,
			)

			if db == nil {
				goto continueInit
			}

			var backend, err = backend.GetBackend(db.Driver())
			if err != nil {
				if errors.Is(err, models.ErrBackendNotFound) {
					registry.backend = NewInMemoryStorageBackend()
					goto continueInit
				}
				return errors.Wrap(err, "Failed to get audit logs backend")
			}

			if err = backend.CreateTable(db); err != nil {
				return errors.Wrap(err, "Failed to create audit logs table")
			}

			qs, err := backend.NewQuerySet(db)
			if err != nil {
				return errors.Wrap(err, "Failed to create audit logs query set")
			}

			registry.backend = qs
		}

	continueInit:
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

				if _, err := Log(hookName, level, instance, data); err != nil {
					logger.Warn(err)
				}
			}),
		)
	}

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

	Logs.Ready = func() error {
		admin.AdminSite.Route.Handle(
			mux.ANY, "/auditlogs", mux.NewHandler(auditLogView),
			"auditlogs",
		)
		return nil
	}

	return Logs
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
	var backend = Backend()
	if backend == nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to setup audit logs backend",
		)
		return
	}

	var logFilters = make([]AuditLogFilter, 0)
	var filterType = r.URL.Query()["type"]
	if len(filterType) > 0 {
		logFilters = append(logFilters, FilterType(filterType...))
	}

	var filterUser = r.URL.Query()["user"]
	if len(filterUser) > 0 {
		var userIds = make([]interface{}, 0, len(filterUser))
		for _, u := range filterUser {
			if isNumber(u) {
				userId, _ := strconv.Atoi(u)
				userIds = append(userIds, userId)
			} else {
				userIds = append(userIds, u)
			}
		}
		logFilters = append(logFilters, FilterUserID(userIds...))
	}

	var filterObjects = r.URL.Query()["object_id"]
	if len(filterObjects) > 0 {
		var objectIds = make([]interface{}, 0, len(filterObjects))
		for _, o := range filterObjects {
			if isNumber(o) {
				objId, _ := strconv.Atoi(o)
				objectIds = append(objectIds, objId)
			} else {
				objectIds = append(objectIds, o)
			}
		}
		logFilters = append(logFilters, FilterObjectID(objectIds...))
	}

	objectPackage := r.URL.Query().Get("content_type")
	if objectPackage != "" {
		var contentType = contenttypes.DefinitionForType(
			objectPackage,
		)
		if contentType != nil {
			logFilters = append(logFilters, FilterContentType(
				contentType.ContentType(),
			))
		}
	}

	var filter = filters.NewFilters[LogEntry](
		"filters",
	)

	filter.Add(&filters.BaseFilterSpec[LogEntry]{
		SpecName:  "type",
		FormField: fields.CharField(),
		Apply: func(value interface{}, objectList []LogEntry) error {
			if fields.IsZero(value) {
				return nil
			}

			var v = value.(string)
			if v != "" {
				logFilters = append(logFilters, FilterType(v))
			}
			return nil
		},
	})

	filter.Add(&filters.BaseFilterSpec[LogEntry]{
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
		Apply: func(value interface{}, objectList []LogEntry) error {
			if fields.IsZero(value) {
				return nil
			}

			var v = value.(string)
			var level, err = strconv.Atoi(v)
			if err != nil {
				return err
			}
			logFilters = append(logFilters, FilterLevelEqual(
				logger.LogLevel(level),
			))
			return nil
		},
	})

	var paginator = pagination.Paginator[LogEntry]{
		GetObject: func(l LogEntry) LogEntry {
			return Define(r, l)
		},
		GetObjects: func(i1, i2 int) ([]LogEntry, error) {
			var (
				objects []LogEntry
				err     error
			)

			if err = filter.Filter(r.URL.Query(), nil); err != nil {
				return nil, errors.Wrap(err, "Failed to filter audit logs")
			}

			if len(logFilters) > 0 {
				objects, err = backend.EntryFilter(
					logFilters, i1, i2,
				)
			} else {
				objects, err = backend.RetrieveMany(
					i1, i2,
				)
			}
			if err != nil {
				return nil, err
			}
			return objects, nil
		},
		GetCount: func() (int, error) {
			if len(logFilters) > 0 {
				return backend.CountFilter(logFilters)
			}
			return backend.Count()
		},
		Amount: 15,
	}

	var pageNum = pagination.GetPageNum(
		r.URL.Query().Get("page"),
	)

	if pageNum < 1 {
		pageNum = 1
	}

	var page, err = paginator.Page(pageNum)
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

	var objectId string
	if len(filterObjects) > 0 {
		objectId = filterObjects[0]
	}

	adminCtx.Set("content_type", objectPackage)
	adminCtx.Set("object_id", objectId)
	adminCtx.Set("log_filters", logFilters)
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
