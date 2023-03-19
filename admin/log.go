package admin

import (
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/admin/internal/paginator"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/db"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/httputils/orderedmap"
	"github.com/Nigel2392/go-django/core/modelutils"
	"github.com/Nigel2392/go-django/core/modelutils/namer"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"gorm.io/gorm"
)

// LogAction is the action that was performed on a model.
type LogAction string

// Predefined actions to log.
const (
	// LogActionCreate is the action that is performed
	// when a model is created.
	LogActionCreate LogAction = "create"
	// LogActionUpdate is the action that is performed
	// when a model is updated.
	LogActionUpdate LogAction = "update"
	// LogActionDelete is the action that is performed
	// when a model is deleted.
	LogActionDelete LogAction = "delete"
	// LogActionUnauthorized is the an action which specifies
	// that a user did not have permission to perform an action.
	LogActionUnauthorized LogAction = "unauthorized"
	// LogActionLoginFailed is the action that is performed
	// when a user fails to login to the admin site.
	LogActionLoginFailed LogAction = "login failed"
	// LogActionLogin is the action that is performed
	// when a user logs into the admin site.
	LogActionLogin LogAction = "login"
)

// LoggableUser is a user that can be logged.
type LoggableUser struct {
	ID       int64 `gorm:"primaryKey;autoIncrement;column:id;type:bigint;"`
	UserID   int64 `gorm:"column:user_id;type:bigint;"`
	Username string
	Email    string
}

// FromUser creates a LoggableUser from a User.
func LogUser(user *auth.User) *LoggableUser {
	var u = &LoggableUser{}
	if user == nil {
		return u
	}
	u.UserID = user.ID
	u.Username = user.Username
	u.Email = user.Email
	return u
}

// String returns the string representation of the user.
func (u *LoggableUser) String() string {
	var a = u.LoginField()
	if a == "" {
		return fmt.Sprintf("%v", u.ID)
	}
	return a
}

// Get the value of the currently set login field.
func (u *LoggableUser) LoginField() string {
	var a, err = modelutils.GetField(u, auth.USER_MODEL_LOGIN_FIELD, true)
	if err != nil {
		return ""
	}
	switch ret := a.(type) {
	case string:
		return ret
	default:
		return fmt.Sprintf("%v", ret)
	}
}

// Log is a record of an action performed on a model
// or on the admin site.
type Log struct {
	gorm.Model
	// The user, if any, that performed the action.
	User *LoggableUser `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	// The ID of the user that performed the action.
	UserID int64
	// The name of the model that the log is for.
	ModelName string
	// Package the model resides in.
	ModelPackage string
	// The ID of the model that the log is for.
	ModelID modelutils.ID
	// The action that was performed on the model.
	Action LogAction
	// The changes that were made to the model.
	Meta *orderedmap.Map[string, any]
}

func (l *Log) WithIP(r *request.Request) *Log {
	l.Meta.Set("ip", r.IP())
	return l
}
func (l *Log) AdminSearch(query string, tx *gorm.DB) *gorm.DB {
	return tx.Where("model_name LIKE ?", fmt.Sprintf("%%%s%%", query)).
		Or("action LIKE ?", fmt.Sprintf("%%%s%%", query))
}

func (l *Log) ActionDisplay() string {
	var b strings.Builder
	var adder string
	switch l.Action {
	case LogActionCreate:
		adder = "created"
	case LogActionUpdate:
		adder = "updated"
	case LogActionDelete:
		adder = "deleted"
	case LogActionUnauthorized:
		b.WriteString("Unauthorized Access")
		if l.User != nil && l.User.LoginField() != "" {
			b.WriteString(" by ")
			b.WriteString(l.User.Username)
		}
		return b.String()
	case LogActionLoginFailed:
		adder = "failed to login"
	case LogActionLogin:
		adder = "logged in"
	}
	if l.User != nil {
		b.WriteString(l.User.LoginField())
		b.WriteString(" ")
		// capitalize the first letter of the adder
	} else {
		if len(adder) > 0 {
			adder = strings.ToUpper(adder[:1]) + adder[1:]
		}
	}
	b.WriteString(adder)
	if l.ModelName != "" {
		b.WriteString(" ")
		b.WriteString(l.ModelName)
	}
	return b.String()
}

func (l *Log) Level() string {
	switch l.Action {
	case LogActionCreate:
		return "success"
	case LogActionUpdate:
		return "info"
	case LogActionDelete:
		return "danger"
	case LogActionUnauthorized:
		return "warning"
	case LogActionLoginFailed:
		return "warning"
	case LogActionLogin:
		return "success"
	}
	return "default"
}

func (l *Log) String() string {
	return fmt.Sprintf("%s %s %s", l.User, l.Action, l.ModelName)
}

// Save the log.
// Errors will be logged to the AdminSite_Logger automatically.
func (l *Log) Save() error {
	var dbItem, err = AdminSite_DB_POOL.ByModel(&Log{})
	if dbItem == nil || err != nil {
		AdminSite_Logger.Critical(err)
		return err
	}
	err = dbItem.DB().Save(l).Error
	if err != nil {
		AdminSite_Logger.Critical(err)
	}
	return err
}

func ModelLog(user *auth.User, model any, action LogAction) *Log {
	var err = user.Refresh()
	if err != nil {
		AdminSite_Logger.Critical(err)
	}

	modelID := modelutils.GetID(model, "ID")
	l := &Log{
		User:         LogUser(user),
		ModelName:    modelutils.GetModelDisplay(model),
		ModelPackage: namer.GetAppName(model),
		ModelID:      modelID,
		Action:       action,
		Meta:         orderedmap.New[string, any](),
	}
	return l
}

func SimpleLog(user *auth.User, action LogAction) *Log {
	l := &Log{
		User:   LogUser(user),
		Action: action,
		Meta:   orderedmap.New[string, any](),
	}
	return l
}

func logView(rq *request.Request) {
	var template, name, err = adminSiteManager.Get("admin/internal/log.tmpl")
	if err != nil {
		rq.Error(500, err.Error())
		return
	}

	var logs = make([]*Log, 0)
	var page, limit, redirected = paginator.PaginateRequest(rq, &Log{}, adminSite.URL(router.GET, "admin:internal:log").Format(), db.GetDefaultDatabase(auth.DB_KEY, AdminSite_DB_POOL).DB(),
		map[string]string{"search": rq.Request.URL.Query().Get("search")})
	if redirected {
		return
	}

	var tx = db.GetDefaultDatabase(auth.DB_KEY, AdminSite_DB_POOL).DB().Order("created_at desc")
	// Get query params
	var searchQuery = rq.Request.URL.Query().Get("search")
	// Search the database
	if searchQuery != "" {
		tx = (&Log{}).AdminSearch(searchQuery, tx)
	}
	// Paginate the results
	tx = paginator.PaginateDB(page, limit)(tx)
	tx.Preload("User")
	tx.Find(&logs)

	rq.Data.Set("logs", logs)
	rq.Data.Set("current_url", rq.Request.URL.String())
	rq.Data.Set("limit_choices", []int{10, 25, 50, 100})
	rq.Data.Set("limit", limit)

	rq.Data.Set("has_search", true)

	err = response.RenderTemplate(rq, template, name)
	if err != nil {
		if rq.Logger != nil {
			rq.Logger.Critical(err)
		}
	}
}

func logGroup() router.Registrar {
	var rt = router.Group("/download", "download")
	rt.Get("", func(r *request.Request) {
		var logs []*Log = make([]*Log, 0)
		db.GetDefaultDatabase(auth.DB_KEY, AdminSite_DB_POOL).DB().Model(Log{}).Preload("User.Groups").Find(&logs)
		var json, err = httputils.Jsonify(logs, 2)
		if err != nil {
			r.Error(500, err.Error())
			return
		}
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.Header().Set("Content-Disposition", "attachment; filename=\"logs.json\"")
		r.Response.Write(json)
	}, "download")
	return rt
}
