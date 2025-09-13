package pages

import (
	"context"
	"fmt"
	"net/http"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/columns"
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
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
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
				return nil
			},
		),
		list.FuncColumn(
			trans.S("Slug"),
			pageRevisionData("Slug"),
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
		ListColumns:   columns,
		DefaultAmount: 25,
		Model:         &revisions.Revision{},

		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/revisions/list.tmpl",
		QuerySet: func(r *http.Request) *queries.QuerySet[*revisions.Revision] {
			return revisions.NewRevisionQuerySet[Page]().
				WithContext(r.Context()).
				ForObjects(p).
				OrderBy("-CreatedAt").
				Base()
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

			breadcrumbs, err := getPageBreadcrumbs(r, p, false)
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

	var fieldDefs = instance.FieldDefs()
	var definition = DefinitionForObject(instance)
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
		panels = definition.EditPanels(r, instance)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](r.Context(), instance)
	var adminForm = admin.NewAdminModelForm[modelforms.ModelForm[attrs.Definer]](form, panels...)
	adminForm.Load()

	if err := r.ParseForm(); err != nil {
		except.Fail(500, err)
		return
	}

	var publishPage = r.FormValue("publish-page") == "publish-page" && permissions.HasObjectPermission(
		r, p, "pages:publish",
	)

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) error {
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
			// if publishing, set the published at time to now
			var err = NewPageQuerySet().
				WithContext(ctx).
				UpdateNode(ref)
			if err != nil {
				return errors.Wrap(err, "failed to update page node")
			}
		}

		// create a new revision if there are changes
		if hasChanged {
			_, err := revisions.CreateDatedRevision(ctx, ref, ref.LatestRevisionCreatedAt)
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

	var view = &views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]]{
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

				var breadcrumbs, err = getPageBreadcrumbs(r, p, false)
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
		GetFormFn: func(req *http.Request) *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer] {
			return adminForm
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range fieldDefs.Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]) {
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
		newInstance, err = p.Specific(r.Context())
		except.Assert(
			err == nil, http.StatusInternalServerError,
			"failed to retrieve specific instance from page: %v", err,
		)
		newChangedTime = p.Reference().LatestRevisionCreatedAt
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
					breadcrumbs, err := getPageBreadcrumbs(r, p, false)
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
