package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/admin/internal/forms"
	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/admin/internal/paginator"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"gorm.io/gorm"
)

func indexView(as *AdminSite, rq *request.Request) {
	var template, name, err = as.templateManager().Get("admin/templates/index.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	var logs []*Log = make([]*Log, 0)
	dbItem, err := as.DBPool.ByModel(&Log{})
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}
	var db = dbItem.DB()
	db = db.Order("created_at DESC")
	db = db.Limit(8)
	db.Preload("User")
	db.Find(&logs)

	rq.Data.Set("logs", logs)

	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func listView(as *AdminSite, mdl *models.Model, rq *request.Request) {
	if !as.checkPermission(rq.User, mdl.Permissions.View(), mdl.Permissions.List()) {
		Unauthorized(as, rq, "You do not have permission to view this page.")
		return
	}
	var template, name, err = as.templateManager().Get("admin/templates/list.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	db, err := as.DBPool.ByModel(mdl.Mdl)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	var searchQuery = rq.Request.URL.Query().Get("search")
	var page, limit, redirected = paginator.PaginateRequest(
		rq,
		mdl.Mdl,
		rq.URL(router.GET, fmt.Sprintf("admin:%s:%s", mdl.AppName(), mdl.Name)).Format(),
		db.DB(),
		map[string]string{"search": searchQuery},
	)
	if redirected {
		return
	}

	var hasSearch bool = false
	var models = mdl.Models(db.DB(), func(tx *gorm.DB) *gorm.DB {
		// Get query params
		// Check if model implements SearchField
		if m, ok := any(mdl.Mdl).(core.AdminSearchField); ok {
			hasSearch = true
			if searchQuery != "" {
				tx = m.AdminSearch(searchQuery, tx)
			}
			rq.Data.Set("search_query", searchQuery)
		}

		// Paginate the results
		tx = paginator.PaginateDB(page, limit)(tx)
		return tx
	})

	var model_data = as.tableDataFromModel(mdl.Mdl, models)
	rq.Data.Set("model", mdl)
	rq.Data.Set("model_data", model_data)
	rq.Data.Set("has_search", hasSearch)
	//	rq.Data.Set("has_fitlers", hasFilters)
	rq.Data.Set("create_url", string(rq.URL(router.GET, fmt.Sprintf("admin:%s:%s:create", mdl.AppName(), mdl.Name)).Format()))

	rq.Data.Set("current_url", rq.Request.URL.String())
	rq.Data.Set("limit_choices", []int{10, 25, 50, 100})
	rq.Data.Set("limit", limit)

	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func detailView(as *AdminSite, mdl *models.Model, rq *request.Request) {
	if !as.checkPermission(rq.User, mdl.Permissions.View()) {
		Unauthorized(as, rq, "You do not have permission to view this page.")
		return
	}
	var template, name, err = as.templateManager().Get("admin/templates/detail.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	db, err := as.DBPool.ByModel(mdl.Mdl)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	var id = modelutils.ID(rq.URLParams.Get("id", ""))

	var m = mdl.New()
	var tx = db.DB().Model(m)
	tx, err = id.Switch(m, "ID", tx)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading model", 500, err)
		return
	}
	err = tx.First(m).Error
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading model", 500, err)
		return
	}

	var form = forms.NewForm("post", string(
		rq.URL(
			router.POST,
			fmt.Sprintf("admin:%s:%s:detail", mdl.AppName(), mdl.Name),
		).Format(id)), rq, db.DB(), m)

	if !as.checkPermission(rq.User, mdl.Permissions.Update()) {
		form.Disable()
	}

	if rq.Method() == "POST" {
		var s, created, err = form.Process(rq, db.DB())
		if err != nil {
			as.Logger.Critical(err)
			goto Template
		} else {
			_ = s
			var log *Log
			if created {
				log = ModelLog(as, rq.User.(*auth.User), form.Model, LogActionCreate)
			} else {
				log = ModelLog(as, rq.User.(*auth.User), form.Model, LogActionUpdate)
				log.Meta.Set("updated_fields", form.UpdatedFields)
			}
			err = log.Save(as)
			if err != nil {
				as.Logger.Critical(err)
			}

			rq.Data.AddMessage("success", "Successfully updated model")
			rq.Redirect(rq.URL(router.GET, fmt.Sprintf("admin:%s:%s:list", mdl.AppName(), mdl.Name)).Format(), 302)
			return
		}
	}
Template:
	rq.Data.Set("model", mdl)
	rq.Data.Set("id", id)
	rq.Data.Set("form", form)

	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func createView(as *AdminSite, mdl *models.Model, rq *request.Request) {
	if !as.checkPermission(rq.User, mdl.Permissions.Create()) {
		Unauthorized(as, rq, "You do not have permission to view this page.")
		return
	}
	var template, name, err = as.templateManager().Get("admin/templates/create.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Could not load template", 500, err)
		return
	}

	db, err := as.DBPool.ByModel(mdl.Mdl)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	var m = mdl.New()
	var form = forms.NewForm("post", string(
		rq.URL(
			router.POST,
			fmt.Sprintf("admin:%s:%s:create", mdl.AppName(), mdl.Name),
		).Format()), rq, db.DB(), m)

	if rq.Method() == "POST" {
		var s, created, err = form.Process(rq, db.DB())
		if err != nil {
			as.Logger.Critical(err)
			rq.Data.AddMessage("error", err.Error())
			goto Template
		} else {
			_ = s
			if err != nil {
				as.Logger.Critical(err)
				renderError(as, rq, "Could not create model", 500, err)
				return
			}
			var log *Log
			if created {
				log = ModelLog(as, rq.User.(*auth.User), form.Model, LogActionCreate)
			} else {
				log = ModelLog(as, rq.User.(*auth.User), form.Model, LogActionUpdate)
				if form.UpdatedFields != nil {
					log.Meta.Set("updated_fields", form.UpdatedFields)
				}
			}
			var url, _, _ = as.getAdminDetailURL(form.Model, modelutils.GetID(form.Model, "ID"))
			log.Meta.Set("url", url)
			log.Save(as)

			rq.Data.AddMessage("success", "Successfully created model")
			rq.Redirect(rq.URL(router.GET, fmt.Sprintf("admin:%s:%s:list", mdl.AppName(), mdl.Name)).Format(), 302)
			return
		}
	}

Template:
	rq.Data.Set("model", mdl)
	rq.Data.Set("form", form)

	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func deleteView(as *AdminSite, mdl *models.Model, rq *request.Request) {
	if !as.checkPermission(rq.User, mdl.Permissions.Delete()) {
		rq.Error(403, "You do not have permission to view this page.")
		return
	}
	var template, name, err = as.templateManager().Get("admin/templates/delete.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	dbItem, err := as.DBPool.ByModel(mdl.Mdl)
	var db = dbItem.DB()
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	var id = modelutils.ID(rq.URLParams.Get("id", ""))
	if id == "" {
		rq.Error(404, "Model not found")
		return
	}

	var m = mdl.New()
	var tx = db.Model(m)
	tx, err = id.Switch(m, "ID", tx)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading model", 500, err)
		return
	}
	err = tx.First(m).Error
	if err != nil {
		rq.Error(500, "Model not found")
		return
	}

	if rq.Method() == "POST" {
		var err = db.Model(mdl.New()).Unscoped().Delete("WHERE id = ?", id).Error
		if err != nil {
			as.Logger.Critical(err)
			renderError(as, rq, "Error deleting model", 500, err)
			return
		}

		var mLog = ModelLog(as, rq.User.(*auth.User), m, LogActionDelete)
		mLog.Meta.Set("ID", id)
		mLog.Save(as)

		rq.Data.AddMessage("success", "Successfully deleted model")
		rq.Redirect(mdl.URL(), 302)
		return
	}

	rq.Data.Set("title", "Delete "+mdl.Name)
	rq.Data.Set("model", mdl)
	rq.Data.Set("instance", m)

	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func unauthorizedView(as *AdminSite, rq *request.Request) {
	//	if hasAdminPerms(rq) {
	//		if rq.Next() != "" {
	//			rq.Redirect(rq.Next(), 302)
	//			return
	//		}
	//		rq.Redirect(AdminSite_URL, 302)
	//		return
	//	}

	var template, name, err = as.templateManager().Get("admin/errors/unauthorized.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	rq.Data.Set("title", "Unauthorized")

	rq.ReSetNext()

	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func loginView(as *AdminSite, rq *request.Request) {
	var template, name, err = as.templateManager().Get("admin/auth/login.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	var form = auth.LoginForm("admin-form-input", "admin-form-label")

	if rq.Method() == "POST" {
		if form.Fill(rq) {
			var login = form.Field(auth.USER_MODEL_LOGIN_FIELD).Value
			var password = form.Field("password").Value
			var user, err = auth.Login(rq, login, password)
			if err != nil {
				// Log failed login
				var log = SimpleLog(auth.NewUser(login), LogActionLoginFailed)
				log.Meta.Set("login", login)
				log.Meta.Set("database", err.Error())
				log.Save(as)

				rq.Data.AddMessage("error", "Login failed")
				as.Logger.Warning(err)
				rq.Redirect(rq.URL(router.GET, "admin:login").Format(), 302)
				return
			} else {
				SimpleLog(user, LogActionLogin).WithIP(rq).Save(as)
				rq.Data.AddMessage("success", "Successfully logged in")
				if rq.Next() != "" {
					rq.Redirect(rq.Next(), 302)
					return
				}
				rq.Redirect(as.URL, 302)
				return
			}
		} else {
			var login = form.Field(auth.USER_MODEL_LOGIN_FIELD).Value
			var u = auth.NewUser(login)
			if err != nil {
				//lint:ignore ST1005 This is a log message
				as.Logger.Critical(fmt.Errorf("Failed to set login field: %s", err.Error()))
				u.Email = login
				u.Username = login
			}
			var log = SimpleLog(u, LogActionLoginFailed)
			log.Meta.Set("login", login)
			log.Meta.Set("error", "Form validation failed")
			for _, err := range form.Errors {
				log.Meta.Set(err.Name, err.FieldErr.Error())
			}
			log.Save(as)
			rq.Data.AddMessage("error", "Login failed")
			rq.Redirect(rq.URL(router.GET, "admin:login").Format(), 302)
			return
		}
	}

	rq.Data.Set("title", "Login")
	rq.Data.Set("form", form)
	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template", 500, err)
	}
}

func logoutView(as *AdminSite, rq *request.Request) {
	var template, name, err = as.templateManager().Get("admin/auth/logout.tmpl")
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error loading template", 500, err)
		return
	}

	if rq.User.IsAuthenticated() {
		err = auth.Logout(rq)
		if err != nil {
			as.Logger.Critical(err)
			renderError(as, rq, "Error logging out", 500, err)
			return
		}
		rq.Data.AddMessage("success", "Successfully logged out")
		rq.Redirect(rq.URL(router.GET, "admin:logout").Format(), 302)
	}

	rq.Data.Set("title", "Logout")
	err = response.Template(rq, template, name)
	if err != nil {
		as.Logger.Critical(err)
		renderError(as, rq, "Error rendering template.", 500, err)
	}
}

func renderError(as *AdminSite, rq *request.Request, err string, code int, errDetail error) {
	rq.Response.Buffer().Reset()
	var template, name, tErr = as.templateManager().Get("admin/errors/errors.tmpl")
	if tErr != nil {
		as.Logger.Critical(tErr)
		rq.Error(500, http.StatusText(500))
		return
	}

	rq.Data.Set("title", "Error")
	rq.Data.Set("error", err)
	rq.Data.Set("error_code", code)
	rq.Data.Set("detail", errDetail.Error())

	tErr = response.Template(rq, template, name)
	if tErr != nil {
		as.Logger.Critical(tErr)
		rq.Error(500, http.StatusText(500))
	}
}
