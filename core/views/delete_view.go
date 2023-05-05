package views

import (
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

type DeleteView[model interfaces.Deleter] struct {
	BaseView[model]
}

func (d *DeleteView[model]) ServeHTTP(r *request.Request) {
	d.BaseView.Action = "delete"
	if d.BaseView.Get == nil {
		d.BaseView.Get = d.get
	}
	if d.BaseView.Post == nil {
		d.BaseView.Post = d.post
	}
	d.BaseView.Serve(r)
}

func (v *DeleteView[T]) get(r *request.Request, data T) {
	var err error
	r.Data.Set("data", &data)
	if v.BackURL != nil {
		r.Data.Set("back_url", v.BackURL(r, data))
	}
	if v.SuccessURL != nil {
		r.Data.Set("success_url", v.SuccessURL(r, data))
	}
	err = response.Render(r, v.Template)
	if err != nil {
		r.Error(500, err.Error())
		return
	}
}

func (v *DeleteView[T]) post(r *request.Request, data T) {
	var err error
	if err != nil {
		r.Data.AddMessage("error", "Error deleting item, please try again later.")
		if v.BackURL != nil {
			r.Redirect(v.BackURL(r, data), 302)
		} else {
			r.Redirect("/", 302)
		}
		return
	}
	r.Data.AddMessage("success", "Item deleted successfully!")
	if v.SuccessURL != nil {
		r.Redirect(v.SuccessURL(r, data), 302)
	} else {
		r.Redirect("/", 302)
	}
	return
}
