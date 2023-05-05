package views

import (
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

type DetailView[T any] struct {
	BaseView[T]

	DeleteURL func(r *request.Request, model T) string
	EditURL   func(r *request.Request, model T) string
}

func (d *DetailView[T]) ServeHTTP(r *request.Request) {
	d.BaseView.Action = "detail"
	if d.BaseView.Get == nil {
		d.BaseView.Get = d.get
	}
	d.BaseView.Serve(r)
}

func (v *DetailView[T]) get(r *request.Request, data T) {
	r.Data.Set("data", data)
	if v.DeleteURL != nil {
		r.Data.Set("delete_url", v.DeleteURL(r, data))
	}
	if v.EditURL != nil {
		r.Data.Set("edit_url", v.EditURL(r, data))
	}
	var err = response.Render(r, v.Template)
	if err != nil {
		r.Error(500, err.Error())
		return
	}
}
