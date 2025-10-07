package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/columns"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/a-h/templ"
	"github.com/alexedwards/scs/v2"
)

func pageRevisionData(fieldName string) func(*http.Request, attrs.Definitions, *revisions.Revision) any {
	return func(r *http.Request, _ attrs.Definitions, instance *revisions.Revision) any {
		var obj, err = instance.AsObject(queries.CommitContext(r.Context(), false))
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return nil
		}

		var p, ok = obj.(Page)
		if !ok {
			except.Fail(http.StatusInternalServerError, errors.TypeMismatch.Wrapf(
				"expected Page interface type, got %T", obj,
			))
			return nil
		}

		ref := p.Reference()
		defs := ref.FieldDefs()
		val := defs.Get(fieldName)
		return val
	}
}

func listRevisionHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:list_revisions") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var columns = []list.ListColumn[*revisions.Revision]{
		list.FuncColumn(
			trans.S("Title"),
			pageRevisionData("Title"),
		),
		list.FuncColumn(
			trans.S("Live"),
			func(r *http.Request, defs attrs.Definitions, rev *revisions.Revision) any {
				if rev.CreatedAt.Equal(p.PublishedAt) {
					return true
				}
				return nil
			},
		),
		list.FuncColumn(
			trans.S("Slug"),
			pageRevisionData("Slug"),
		),
		list.FuncColumn(
			trans.S("User"),
			func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) interface{} {
				if row.User == nil {
					return nil
				}
				return attrs.ToString(row.User)
			},
		),
		&columns.ListActionsColumn[*revisions.Revision]{
			Heading: trans.S("Actions"),
			Actions: []*columns.ListAction[*revisions.Revision]{
				{
					Text: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) string {
						return trans.T(r.Context(), "Compare to current")
					},
					URL: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) string {
						var next = django.Reverse(
							"admin:pages:revisions",
							p.PK,
						)
						return addNextUrl(
							django.Reverse("admin:pages:revisions:compare", p.PK, row.ID),
							next,
						)
					},
				},
				{
					Text: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) string {
						return trans.T(r.Context(), "Compare to")
					},
					Attrs: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) templ.Attributes {
						var m = make(templ.Attributes)
						m["data-controller"] = "pages-revision-compare"
						m["data-pages-revision-compare-list-url-value"] = fmt.Sprintf(
							"%s?page-id=%d",
							django.Reverse(
								"admin:apps:model:chooser:list",
								"revisions",
								"revision",
								CHOOSER_PAGE_REVISIONS_KEY,
							),
							p.PK,
						)
						m["data-pages-revision-compare-current-id-value"] = row.ID
						m["data-pages-revision-compare-url-value"] = addNextUrl(
							django.Reverse("admin:pages:revisions:compare_to", p.PK, "__OLD_ID__", "__NEW_ID__"),
							r.URL.String(),
						)
						return m
					},
				},
				{
					Text: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) string {
						return trans.T(r.Context(), "Preview")
					},
					URL: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) string {
						return django.Reverse("admin:pages:revisions:revision_preview", p.PK, row.ID)
					},
					Attrs: func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) templ.Attributes {
						var m = make(templ.Attributes)
						m["target"] = "_blank"
						m["rel"] = "noreferrer noopener"
						return m
					},
				},
			},
		},
		columns.TimeSinceColumn[*revisions.Revision](
			trans.S("Created"),
			"CreatedAt",
			trans.LONG_TIME_FORMAT,
		),
	}

	var next = django.Reverse(
		"admin:pages:revisions",
		p.ID(),
	)

	var view = &list.View[*revisions.Revision]{
		ListColumns:     columns,
		DefaultAmount:   25,
		Model:           &revisions.Revision{},
		PageParam:       "page",
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/revisions/list.tmpl",
		QuerySet: func(r *http.Request) *queries.QuerySet[*revisions.Revision] {
			return revisions.NewRevisionQuerySet().
				WithContext(r.Context()).
				ForObjects(p).
				OrderBy("-CreatedAt").
				Base().
				SelectRelated("User")
		},
		ChangeContextFn: func(req *http.Request, qs *queries.QuerySet[*revisions.Revision], viewCtx ctx.Context) (ctx.Context, error) {
			var context = admin.NewContext(
				req, admin.AdminSite, viewCtx,
			)

			var contentType *contenttypes.ContentTypeDefinition
			if p.ContentType != "" {
				contentType = contenttypes.DefinitionForType(
					p.ContentType,
				)
			} else {
				contentType = contenttypes.DefinitionForObject(
					p,
				)
			}

			context.Set("app", a)
			context.Set("model", m)
			context.Set("page_object", p)
			context.Set(
				"model_name",
				contentType.Label(r.Context()),
			)

			var paginator = list.PaginatorFromContext[*revisions.Revision](req.Context())
			var count, err = paginator.Count()
			if err != nil {
				return nil, err
			}

			breadcrumbs, err := getPageBreadcrumbs(r, p, true)
			if err != nil {
				return nil, err
			}

			breadcrumbs = append(breadcrumbs, admin.BreadCrumb{
				Title: trans.T(r.Context(), "Revisions"),
				URL:   "",
			})

			context.SetPage(admin.PageOptions{
				BreadCrumbs: breadcrumbs,
				Actions:     getPageActions(r, p),
				TitleFn: trans.SP(
					"%s %q (%d revision)",
					"%s %q (%d revisions)",
					count,
					contentType.Label(r.Context()),
					p.Title, count,
				),
			})

			return context, nil
		},
		TitleFieldColumn: func(col list.ListColumn[*revisions.Revision]) list.ListColumn[*revisions.Revision] {
			return list.TitleFieldColumn(
				col,
				func(r *http.Request, defs attrs.Definitions, row *revisions.Revision) string {
					if row.CreatedAt.Equal(p.PublishedAt) {
						return ""
					}
					return addNextUrl(django.Reverse(
						"admin:pages:revisions:detail",
						p.PK, row.ID),
						next,
					)
				},
			)
		},
	}

	views.Invoke(view, w, r)
}

func revisionDetailHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var vars = mux.Vars(r)
	var revisionID = vars.GetInt("revision_id")
	if revisionID == 0 {
		except.Fail(http.StatusBadRequest, "invalid revision ID: %v", vars.Get("revision_id"))
		return
	}

	var chosenRevision, err = revisions.GetRevisionByID(r.Context(), int64(revisionID))
	except.Assert(
		err == nil, http.StatusInternalServerError,
		"failed to retrieve latest revision for page: %v", err,
	)

	instance, err := (*revisions.TypedRevision[Page])(chosenRevision).AsObject(r.Context())
	except.Assert(
		err == nil, http.StatusInternalServerError,
		"failed to retrieve specific instance from revision: %v", err,
	)

	var adminForm = PageEditForm(r, instance)
	adminForm.Load()

	if err := r.ParseForm(); err != nil {
		except.Fail(500, err)
		return
	}

	var publishPage = r.FormValue("publish-page") == "publish-page" && permissions.HasObjectPermission(
		r, p, "pages:publish",
	)

	adminForm.Form.SaveInstance = func(ctx context.Context, d attrs.Definer) error {
		var hasChanged = adminForm.HasChanged()
		if !hasChanged && !publishPage {
			logger.Warnf("No changes detected for page: %s", instance.Reference().Title)
			return nil
		}

		var wasPublished bool
		var ref = instance.Reference()
		if publishPage && !ref.StatusFlags.Is(StatusFlagPublished) {
			ref.StatusFlags |= StatusFlagPublished
			wasPublished = true
		}

		switch page := d.(type) {
		case *PageNode:
			ref = page
		case Page:
			ref.PageObject = page
		default:
			return fmt.Errorf("invalid page type: %T", d)
		}

		ref.LatestRevisionCreatedAt = chosenRevision.CreatedAt
		if publishPage {
			ref.PublishedAt = chosenRevision.CreatedAt
		}

		if publishPage {
			// if publishing, set the published at time to now
			var err = NewPageQuerySet().
				WithContext(ctx).
				UpdateNode(ref)
			if err != nil {
				return errors.Wrap(err, "failed to update page node")
			}
		} else {
			// if not publishing, just save the page object without changing the published at time
			var err = NewPageQuerySet().
				WithContext(ctx).
				ExplicitSave().
				Select("LatestRevisionCreatedAt").
				updateNode(ref)
			if err != nil {
				return errors.Wrap(err, "failed to update page")
			}
		}

		// create a new revision if there are changes
		if hasChanged {
			_, err := revisions.CreateDatedRevision(ctx, ref, authentication.Retrieve(r).(users.User), ref.LatestRevisionCreatedAt)
			if err != nil {
				return errors.Wrap(err, "failed to create new revision")
			}
		}

		var logAction string
		var dataMap = map[string]interface{}{
			"page_id": instance.ID(),
			"label":   ref.Title,
			"cType":   p.ContentType,
		}

		switch {
		case wasPublished:
			logAction = "pages:publish_revision"
			dataMap["edited"] = true
		default:
			logAction = "pages:revert_revision"
		}

		auditlogs.Log(ctx, logAction, logger.INF, p, dataMap)

		return nil
	}

	var view = &views.FormView[*admin.AdminForm[*modelforms.BaseModelForm[attrs.Definer], attrs.Definer]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/revisions/edit.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)
				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", instance)
				context.Set("is_published", p.StatusFlags.Is(StatusFlagPublished))

				var backURL string
				if q := req.FormValue("next"); q != "" {
					backURL = q
				}
				context.Set("BackURL", backURL)
				context.Set("PostURL", django.Reverse(
					"admin:pages:revisions:detail", p.ID(), chosenRevision.ID,
				))

				var breadcrumbs, err = getPageBreadcrumbs(r, p, true)
				if err != nil {
					return nil, err
				}

				breadcrumbs = append(breadcrumbs,
					admin.BreadCrumb{
						Title: trans.T(r.Context(), "Revisions"),
						URL:   django.Reverse("admin:pages:revisions", p.ID()),
					},
					admin.BreadCrumb{
						Title: trans.T(r.Context(), "Edit revision for %q", instance.Reference().Title),
						URL:   "",
					},
				)

				context.SetPage(admin.PageOptions{
					TitleFn:     trans.S("Edit revision for %q", instance.Reference().Title),
					BreadCrumbs: breadcrumbs,
					Actions:     getPageActions(r, p),
				})

				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminForm[*modelforms.BaseModelForm[attrs.Definer], attrs.Definer] {
			return adminForm
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			var fieldDefs = instance.FieldDefs()
			for _, field := range fieldDefs.Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminForm[*modelforms.BaseModelForm[attrs.Definer], attrs.Definer]) {
			var instance = form.Instance()
			assert.False(instance == nil, "instance is nil after form submission")
			var page = instance.(Page)
			var ref = page.Reference()

			messages.Success(r, "Revision restored successfully")

			http.Redirect(w, r, django.Reverse("admin:pages:list", ref.ID()), http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}

func revisionCompareHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var vars = mux.Vars(r)
	var revisionID = vars.GetInt("revision_id")
	if revisionID == 0 {
		except.Fail(http.StatusBadRequest, "invalid revision ID: %v", vars.Get("revision_id"))
		return
	}

	var chosenRevision, err = revisions.GetRevisionByID(r.Context(), int64(revisionID))
	except.Assert(
		err == nil, http.StatusInternalServerError,
		"failed to retrieve latest revision for page: %v", err,
	)

	oldInstance, err := (*revisions.TypedRevision[Page])(chosenRevision).AsObject(r.Context())
	except.Assert(
		err == nil, http.StatusInternalServerError,
		"failed to retrieve specific instance from revision: %v", err,
	)

	var newInstance Page
	var newChangedTime time.Time
	var otherRevisionID = vars.GetInt("other_revision_id")
	if otherRevisionID == 0 {
		var latestRevisionRow, err = revisions.NewRevisionQuerySet().
			WithContext(r.Context()).
			ForObjects(p).
			Filter("CreatedAt", p.LatestRevisionCreatedAt).
			OrderBy("-CreatedAt").
			Get()
		except.Assert(
			err == nil, http.StatusInternalServerError,
			"failed to retrieve latest revision for page: %v", err,
		)

		var latestRevision = latestRevisionRow.Object
		newInstance, err = (*revisions.TypedRevision[Page])(latestRevision).AsObject(r.Context())
		except.Assert(
			err == nil, http.StatusInternalServerError,
			"failed to retrieve specific instance from latest revision: %v", err,
		)

		newChangedTime = latestRevision.CreatedAt

	} else {
		var otherRevision, err = revisions.GetRevisionByID(r.Context(), int64(otherRevisionID))
		except.Assert(
			err == nil, http.StatusInternalServerError,
			"failed to retrieve other revision for page: %v", err,
		)
		newInstance, err = (*revisions.TypedRevision[Page])(otherRevision).AsObject(r.Context())
		except.Assert(
			err == nil, http.StatusInternalServerError,
			"failed to retrieve specific instance from other revision: %v", err,
		)
		newChangedTime = otherRevision.CreatedAt
	}

	var fieldDefs = oldInstance.FieldDefs()
	var definition = DefinitionForObject(oldInstance)
	var panels []admin.Panel
	if definition == nil || definition.EditPanels == nil {
		panels = make([]admin.Panel, 0, fieldDefs.Len())
		for _, def := range fieldDefs.Fields() {
			var formField = def.FormField()
			if formField == nil {
				continue
			}

			panels = append(panels, admin.FieldPanel(def.Name()))
		}
	} else {
		panels = definition.EditPanels(r, oldInstance)
	}

	comparisonClass, err := admin.PanelComparison(r.Context(), panels, oldInstance, newInstance, true)
	except.Assert(
		err == nil, http.StatusInternalServerError,
		"failed to create comparison for revision: %v", err,
	)

	var view = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/revisions/compare.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)
			context.Set("app", a)
			context.Set("model", m)
			context.Set("old_instance", oldInstance)
			context.Set("new_instance", newInstance)
			context.Set("page_object", p)
			context.Set("comparison", comparisonClass)

			var backURL string
			if q := req.FormValue("next"); q != "" {
				backURL = q
			}
			context.Set("BackURL", backURL)

			context.SetPage(admin.PageOptions{
				TitleFn: trans.S("Comparing revisions for %q", p.Title),
				SubtitleFn: trans.S(
					"Comparing revision from %s to %s",
					trans.Time(r.Context(), chosenRevision.CreatedAt, trans.LONG_TIME_FORMAT),
					trans.Time(r.Context(), newChangedTime, trans.LONG_TIME_FORMAT),
				),
				BreadCrumbs: func() []admin.BreadCrumb {
					breadcrumbs, err := getPageBreadcrumbs(r, p, true)
					if err != nil {
						except.Fail(http.StatusInternalServerError, err)
						return nil
					}
					return append(breadcrumbs,
						admin.BreadCrumb{
							Title: trans.T(r.Context(), "Revisions"),
							URL:   django.Reverse("admin:pages:revisions", p.PK),
						},
						admin.BreadCrumb{
							Title: trans.T(r.Context(), "Comparing revisions for %q", p.Title),
							URL:   "",
						},
					)
				}(),
				Actions: getPageActions(r, p),
			})
			return context, nil
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(http.StatusInternalServerError, err)
		return
	}
}

type previewContextKey struct{}

var previewContextKeyInstance = previewContextKey{}

func IsPreviewContext(ctx context.Context) bool {
	_, ok := ctx.Value(previewContextKeyInstance).(struct{})
	return ok
}

var PagePreviewHandler = &GenericPreviewHandler[Page]{
	Model:            &PageNode{},
	SessionKeyPrefix: "page_preview_",
	URLKey:           PageIDVariableName,
	Expiration:       10 * time.Minute,
	GetFormFunc: func(h *GenericPreviewHandler[Page], r *http.Request, instance Page, oldData url.Values) (forms.Form, error) {
		return PageEditForm(r, instance), nil
	},
	ServePreview: func(h *GenericPreviewHandler[Page], w http.ResponseWriter, original, previewReq *http.Request, instance Page) {
		servePageView(w, previewReq, instance)
	},
	BuildPreviewRequest: func(h *GenericPreviewHandler[Page], r *http.Request, instance Page) (*http.Request, error) {
		var cloned = r.Clone(context.WithValue(
			r.Context(), previewContextKey{}, struct{}{},
		))
		cloned.URL.Path = URLPath(instance)
		return cloned, nil
	},
	GetObjectFunc: func(h *GenericPreviewHandler[Page], r *http.Request, pk string) (Page, error) {
		var objRow, err = NewPageQuerySet().
			WithContext(r.Context()).
			Filter("PK", pk).
			Get()
		if err != nil {
			return nil, fmt.Errorf("failed to get page with ID %v: %w", pk, err)
		}

		specific, err := Specific(r.Context(), objRow.Object, true)
		if err != nil {
			return nil, fmt.Errorf("failed to get specific page type for page ID %v: %w", pk, err)
		}

		return specific, nil
	},
}

var PageRevisionPreviewHandler = &GenericPreviewHandler[Page]{
	Model:            &PageNode{},
	SessionKeyPrefix: "page_revision_preview_",
	URLKey:           PageIDVariableName,
	Expiration:       10 * time.Minute,
	BuildPreviewRequest: func(h *GenericPreviewHandler[Page], r *http.Request, instance Page) (*http.Request, error) {
		var cloned = r.Clone(context.WithValue(
			r.Context(), previewContextKey{}, struct{}{},
		))
		cloned.URL.Path = URLPath(instance)
		return cloned, nil
	},
	GetFormFunc: func(h *GenericPreviewHandler[Page], r *http.Request, instance Page, oldData url.Values) (forms.Form, error) {
		return PageEditForm(r, instance), nil
	},
	ServePreview: func(h *GenericPreviewHandler[Page], w http.ResponseWriter, original, previewReq *http.Request, instance Page) {
		servePageView(w, previewReq, instance)
	},
	GetObjectFunc: func(h *GenericPreviewHandler[Page], r *http.Request, pk string) (Page, error) {
		var vars = mux.Vars(r)
		var revisionID = vars.GetInt("revision_id")
		var chosenRevision *revisions.Revision
		var objRow, err = NewPageQuerySet().
			WithContext(r.Context()).
			Filter("PK", pk).
			Get()
		if err != nil {
			return nil, fmt.Errorf("failed to get page with ID %v: %w", pk, err)
		}

		if revisionID != 0 {
			chosenRevision, err = revisions.GetRevisionByID(r.Context(), int64(revisionID))
		} else {
			chosenRevision, err = revisions.LatestRevision(r.Context(), objRow.Object)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get latest revision for page ID %v: %w", pk, err)
		}

		specific, err := Specific(r.Context(), objRow.Object.Reference(), true)
		if err != nil {
			return nil, fmt.Errorf("failed to get specific page type for page ID %v: %w", pk, err)
		}

		err = revisions.UnmarshalRevisionData(specific, []byte(chosenRevision.Data))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal latest revision data for page ID %v: %w", pk, err)
		}

		return specific, nil
	},
}

type genericPreviewView interface {
	GetObject(r *http.Request) (attrs.Definer, error)
	GetForm(r *http.Request, instance attrs.Definer, oldData url.Values) (forms.Form, error)
	ValidateForm(r *http.Request, form forms.Form) error
	GetPreviewData(r *http.Request) (url.Values, error)
	SetPreviewData(r *http.Request, data url.Values) error
	DeletePreviewData(r *http.Request) error
	RemoveOldPreviewData(r *http.Request) error
	BuildPreviewRequest(r *http.Request, instance attrs.Definer) (*http.Request, error)
	MakePreviewRequest(w http.ResponseWriter, original *http.Request, previewReq *http.Request, instance attrs.Definer)
}

var (
	_ genericPreviewView = &GenericBoundPreview[attrs.Definer]{}
	_ views.BindableView = &GenericPreviewHandler[attrs.Definer]{}
)

type sessionData struct {
	Data   url.Values
	Stored time.Time
}

type GenericBoundPreview[T attrs.Definer] struct {
	Handler     *GenericPreviewHandler[T]
	ContentType *contenttypes.ContentTypeDefinition
	Object      T
}

func (v *GenericBoundPreview[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (v *GenericBoundPreview[T]) GetObject(r *http.Request) (attrs.Definer, error) {
	if !django_reflect.IsZero(v.Object) {
		return v.Object, nil
	}

	var vars = mux.Vars(r)
	var pk = vars.Get(v.Handler.PathKey())
	if pk == "" {
		return nil, errors.ValueError.Wrap("invalid object ID")
	}

	if v.Handler.GetObjectFunc != nil {
		return v.Handler.GetObjectFunc(v.Handler, r, pk)
	}

	var meta = attrs.GetModelMeta(v.Handler.Model)
	var defs = meta.Definitions()
	var primary = defs.Primary()

	var querySet *queries.QuerySet[T]
	if v.Handler.GetObjectQuerySet != nil {
		querySet = v.Handler.GetObjectQuerySet(v.Handler, r)
	} else {
		querySet = queries.GetQuerySetWithContext(r.Context(), v.Handler.Model)
	}

	querySet = querySet.Filter(primary.Name(), pk)
	objRow, err := querySet.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s with ID %v: %w", v.ContentType.Label(r.Context()), pk, err)
	}

	v.Object = objRow.Object
	return v.Object, nil
}

func (v *GenericBoundPreview[T]) GetForm(r *http.Request, instance attrs.Definer, oldData url.Values) (forms.Form, error) {
	if v.Handler.GetFormFunc != nil {
		return v.Handler.GetFormFunc(v.Handler, r, instance.(T), oldData)
	}

	var form = modelforms.NewBaseModelForm[T](
		r.Context(),
		instance.(T),
		forms.WithRequestData(http.MethodPost, r),
	)

	form.Load()

	return form, nil
}

func (v *GenericBoundPreview[T]) ValidateForm(r *http.Request, form forms.Form) error {
	if !forms.IsValid(r.Context(), form) {
		return fmt.Errorf("form is not valid")
	}
	return nil
}

func (v *GenericBoundPreview[T]) GetPreviewData(r *http.Request) (url.Values, error) {
	var sessionKey = v.Handler.SessionKey(v.Object)
	var session = sessions.Retrieve(r)
	var data, ok = session.Get(sessionKey).(sessionData)
	if !ok {
		return nil, nil
	}
	if data.Stored.Add(v.Handler.Expiration).Before(time.Now()) {
		session.Delete(sessionKey)
		return nil, nil
	}
	return data.Data, nil
}

func (v *GenericBoundPreview[T]) SetPreviewData(r *http.Request, data url.Values) error {
	var sessionKey = v.Handler.SessionKey(v.Object)
	var session = sessions.Retrieve(r)
	session.Set(sessionKey, sessionData{
		Data:   data,
		Stored: time.Now(),
	})
	return nil
}

func (v *GenericBoundPreview[T]) DeletePreviewData(r *http.Request) error {
	var sessionKey = v.Handler.SessionKey(v.Object)
	var session = sessions.Retrieve(r)
	session.Delete(sessionKey)
	return nil
}

func (v *GenericBoundPreview[T]) RemoveOldPreviewData(r *http.Request) error {
	var session = sessions.Retrieve(r)
	var prefix = v.Handler.SessionKeyPrefix
	if prefix == "" {
		prefix = "generic_preview_"
	}

	storeDef, ok := session.(interface{ Store() *scs.SessionManager })
	if !ok {
		return errors.TypeMismatch.Wrapf("session does not support Store(): %T", session)
	}

	go django.Task("Remove old preview data", func(a *django.Application) error {
		var keys = storeDef.Store().Keys(r.Context())
		for _, key := range keys {
			if !strings.HasPrefix(key, prefix) {
				continue
			}

			var data, ok = session.Get(key).(sessionData)
			if !ok {
				continue
			}

			if time.Now().After(data.Stored.Add(v.Handler.Expiration)) {
				session.Delete(key)
			}
		}
		return nil
	})

	return nil
}

func (v *GenericBoundPreview[T]) BuildPreviewRequest(r *http.Request, instance attrs.Definer) (*http.Request, error) {
	if v.Handler.BuildPreviewRequest != nil {
		return v.Handler.BuildPreviewRequest(v.Handler, r, instance.(T))
	}
	var cloned = r.Clone(context.WithValue(
		r.Context(), previewContextKey{}, struct{}{},
	))
	return cloned, nil
}

func (v *GenericBoundPreview[T]) MakePreviewRequest(w http.ResponseWriter, original *http.Request, previewReq *http.Request, instance attrs.Definer) {
	if handler, ok := any(v.Object).(http.Handler); ok {
		handler.ServeHTTP(w, previewReq)
		return
	}

	if v.Handler.ServePreview != nil {
		v.Handler.ServePreview(v.Handler, w, original, previewReq, instance.(T))
		return
	}

	except.Fail(
		http.StatusInternalServerError,
		"object of type %T is not an http.Handler",
		v.Object,
	)
}

type GenericPreviewHandler[T attrs.Definer] struct {
	Model               T
	URLKey              string
	Expiration          time.Duration
	SessionKeyPrefix    string
	GetObjectQuerySet   func(h *GenericPreviewHandler[T], r *http.Request) *queries.QuerySet[T]
	GetObjectFunc       func(h *GenericPreviewHandler[T], r *http.Request, pk string) (T, error)
	GetFormFunc         func(h *GenericPreviewHandler[T], r *http.Request, instance T, oldData url.Values) (forms.Form, error)
	BuildPreviewRequest func(h *GenericPreviewHandler[T], r *http.Request, instance T) (*http.Request, error)
	ServePreview        func(h *GenericPreviewHandler[T], w http.ResponseWriter, original *http.Request, previewReq *http.Request, instance T)
}

func (v *GenericPreviewHandler[T]) SessionKey(obj attrs.Definer) string {
	var prefix = v.SessionKeyPrefix
	if prefix == "" {
		prefix = "generic_preview_"
	}
	var cType = contenttypes.NewContentType(obj)
	return fmt.Sprintf(
		"%s%s_%s_%v",
		prefix,
		cType.AppLabel(),
		cType.Model(),
		attrs.PrimaryKey(obj),
	)
}

func (v *GenericPreviewHandler[T]) PathKey() string {
	if v.URLKey != "" {
		return v.URLKey
	}
	return "object_id"
}

func (v *GenericPreviewHandler[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var bound = &GenericBoundPreview[T]{
		Handler:     v,
		ContentType: contenttypes.DefinitionForObject(v.Model),
	}

	if bound.ContentType == nil {
		return nil, fmt.Errorf("no content type found for model %T", v.Model)
	}

	if obj, err := bound.GetObject(req); err != nil {
		return nil, err
	} else {
		bound.Object = obj.(T)
	}

	return bound, nil
}

/*
Handle embedders of GenericPreviewHandler that also implement genericPreviewView.

This ensures that the correct methods are called on the embedder if they are available.
*/
func (v *GenericPreviewHandler[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (v *GenericPreviewHandler[T]) Methods() []string {
	return []string{http.MethodGet, http.MethodPost, http.MethodDelete}
}

func (v *GenericPreviewHandler[T]) onError(w http.ResponseWriter, r *http.Request, err error, status int) {
	logger.Errorf(
		"Error while serving view: %v", err,
	)
	except.Fail(
		status,
		"Error while serving view: %v", err,
	)
}

func (v *GenericPreviewHandler[T]) onErrorJSON(w http.ResponseWriter, r *http.Request, err error, status int) {
	logger.Errorf(
		"Error while serving view: %v", err,
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var resp = map[string]any{
		"error": err.Error(),
	}
	var enc = json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Errorf("Failed to encode JSON response: %v", err)
	}
}

func (v *GenericPreviewHandler[T]) TakeControl(w http.ResponseWriter, req *http.Request, view views.View) {
	if views.MethodServe(w, req, view) {
		// Any matching serve method takes precedence over the fallback.
		return
	}

	viewObj, ok := view.(genericPreviewView)
	if !ok {
		except.Fail(http.StatusInternalServerError, "Invalid view type")
		return
	}

	// Handle DELETE requests to clear any existing preview data.
	if req.Method == http.MethodDelete {
		err := viewObj.DeletePreviewData(req)
		var resp = map[string]any{
			"success": err == nil,
		}
		if err != nil {
			resp["error"] = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var enc = json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			logger.Errorf("Failed to encode JSON response: %v", err)
		}
		return
	}

	// Handle POST requests to validate and store preview data.
	// Said preview data can then be used in a subsequent GET request
	// to build a preview of the object.
	if req.Method == http.MethodPost {
		instance, err := viewObj.GetObject(req)
		if err != nil {
			v.onErrorJSON(w, req, err, http.StatusInternalServerError)
			return
		}

		if err := viewObj.RemoveOldPreviewData(req); err != nil {
			v.onErrorJSON(w, req, err, http.StatusInternalServerError)
			return
		}

		if err := req.ParseForm(); err != nil {
			v.onErrorJSON(w, req, err, http.StatusInternalServerError)
			return
		}

		form, err := viewObj.GetForm(req, instance, req.PostForm)
		if err != nil {
			v.onErrorJSON(w, req, err, http.StatusInternalServerError)
			return
		}

		var (
			isValid     = true
			isAvailable = true
		)
		if err := viewObj.ValidateForm(req, form); err != nil {
			isValid = false
		}

		if isValid {
			var data, _ = form.Data()
			if err := viewObj.SetPreviewData(req, data); err != nil {
				v.onErrorJSON(w, req, err, http.StatusInternalServerError)
				return
			}
		} else {
			var old, err = viewObj.GetPreviewData(req)
			if err != nil {
				v.onErrorJSON(w, req, err, http.StatusInternalServerError)
				return
			}

			form, err = viewObj.GetForm(req, instance, old)
			if err != nil {
				v.onErrorJSON(w, req, err, http.StatusInternalServerError)
				return
			}

			if err := viewObj.ValidateForm(req, form); err != nil {
				isAvailable = false
			}
		}

		jsonData, err := json.Marshal(map[string]any{
			"is_valid":     isValid,
			"is_available": isAvailable,
		})
		if err != nil {
			v.onErrorJSON(w, req, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
		return
	}

	instance, err := viewObj.GetObject(req)
	if err != nil {
		v.onError(w, req, err, http.StatusInternalServerError)
		return
	}

	// Once the POST request has been made, preview data should be available.
	//
	// This means we can now actually build the preview request and proxy
	// the response back to the user.
	previewData, err := viewObj.GetPreviewData(req)
	if err != nil {
		v.onError(w, req, err, http.StatusInternalServerError)
		return
	}

	// Even though no preview data is available, we can still serve
	// a preview of the current object state.
	// Log a warning and continue.
	if previewData == nil {
		logger.Warnf("no preview data found for object %T with ID %v", instance, attrs.PrimaryKey(instance))
		servePreviewRequest(w, req, v, viewObj, instance)
		return
	}

	form, err := viewObj.GetForm(req, instance, previewData)
	if err != nil {
		v.onError(w, req, err, http.StatusInternalServerError)
		return
	}

	// Make sure none of the form data actually gets saved to the database.
	form.WithContext(queries.CommitContext(
		req.Context(),
		false,
	))

	// Validate the form again before building the preview request.
	if err := viewObj.ValidateForm(req, form); err != nil {
		v.onError(w, req, err, http.StatusBadRequest)
		return
	}

	// If the form has a proper save method, ensure it gets called so that any
	// side effects (like m2m saving) are executed.
	// This will try to find Save methods with the following signatures:
	//   - func() error
	//   - func(ctx context.Context) error
	//   - func(ctx context.Context) (..., error)
	saveFn, err := django_reflect.Method[func() error](form, "Save", django_reflect.WrapWithContext(
		form.Context(),
	))
	if err == nil {
		if err := saveFn(); err != nil {
			v.onError(w, req, err, http.StatusInternalServerError)
			return
		}
	} else {
		logger.Warnf("form does not have Save method for previews: %v", err)
	}

	servePreviewRequest(w, req, v, viewObj, instance)
}

func servePreviewRequest[T attrs.Definer](w http.ResponseWriter, req *http.Request, v *GenericPreviewHandler[T], viewObj genericPreviewView, instance attrs.Definer) {
	previewReq, err := viewObj.BuildPreviewRequest(req, instance)
	if err != nil {
		v.onError(w, req, err, http.StatusInternalServerError)
		return
	}

	viewObj.MakePreviewRequest(w, req, previewReq, instance)
}
