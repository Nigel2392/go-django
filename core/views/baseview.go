package views

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Nigel2392/router/v3/request"
)

type BaseView[MODEL any] struct {
	// The template to use
	Template string

	// Function to get a template with
	GetTemplate func(string) (*template.Template, string, error)

	// The URL to redirect to after a failed action.
	BackURL func(r *request.Request, model MODEL) string

	// The URL to redirect to after a successful action.
	SuccessURL func(r *request.Request, model MODEL) string

	// The permissions required to perform an action on the model
	//
	// If there is a permission registered, the user must be authenticated!
	RequiredPerms []string

	// NeedsAuth specifies whether the user needs to be authenticated to view the page.
	NeedsAuth bool

	// Needs admin permissions to perform an action on the model.
	NeedsAdmin bool

	// Whether a superuser can perform the action.
	SuperUserCanPerform bool

	// The action to perform on the model.
	//
	// This is used to display the correct message to the user.
	//
	// For example: "delete", "update" or "view"
	Action string

	GetQuerySet func(r *request.Request) (MODEL, error)

	// Overrides for the get and post functions.
	Get  func(r *request.Request, m MODEL)
	Post func(r *request.Request, m MODEL)

	// Extra func to run on the data.
	//
	// This can be used for validation, returning an error will make the request error.
	//
	// With the error message.
	Extra func(r *request.Request, model MODEL) error

	// Extra auth func to run on the data.
	//
	// This can be used for validation, returning an error will make the request error.
	//
	// With the error message.
	ExtraAuth func(r *request.Request, model MODEL) error
}

func (v *BaseView[T]) Serve(r *request.Request) {
	var err error
	var data T
	if v.GetQuerySet != nil {
		data, err = v.GetQuerySet(r)
		if err != nil {
			r.Data.AddMessage("error", err.Error())
			goto lastPage
		}
	}
	if v.NeedsAuth {
		if !r.User.IsAuthenticated() && v.NeedsAuth {
			r.Data.AddMessage("error", fmt.Sprintf("You need to be logged in to %s this item.", v.Action))
			goto lastPage
		}

		if !r.User.HasPermissions(v.RequiredPerms...) {
			r.Data.AddMessage("error", fmt.Sprintf("You do not have permission to %s this item.", v.Action))
			goto lastPage
		}

		if v.ExtraAuth != nil {
			err = v.ExtraAuth(r, data)
			if err != nil {
				r.Data.AddMessage("error", err.Error())
				goto lastPage
			}
		}
	}
	switch r.Method() {
	case http.MethodPost:
		if v.Post == nil {
			goto methodNotAllowed
		}
		if v.Extra != nil {
			err = v.Extra(r, data)
			if err != nil {
				r.Data.AddMessage("error", err.Error())
				goto lastPage
			}
		}
		v.Post(r, data)
		return
	case http.MethodGet:
		if v.Get == nil {
			goto methodNotAllowed
		}
		if v.Extra != nil {
			err = v.Extra(r, data)
			if err != nil {
				r.Data.AddMessage("error", err.Error())
				goto lastPage
			}
		}
		v.Get(r, data)
		return
	}
methodNotAllowed:
	r.Data.AddMessage("error", "Method not allowed.")
lastPage:
	if v.BackURL != nil {
		r.Redirect(v.BackURL(r, data), 302)
	} else {
		r.Redirect("/", 302)
	}
}
