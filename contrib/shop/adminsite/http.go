package adminsite

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/views"
)

type ShopHandler struct {
	Handle func(w http.ResponseWriter, r *http.Request)
}

func (s ShopHandler) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (s ShopHandler) TakeControl(w http.ResponseWriter, r *http.Request, _ views.View) {
	s.Handle(w, r)
}

type ShopInstanceHandler[T attrs.Definer] struct {
	Instance T
	Handle   func(w http.ResponseWriter, r *http.Request, instance T)
}

func (s ShopInstanceHandler[T]) ServeXXX(w http.ResponseWriter, r *http.Request) {}

func (s ShopInstanceHandler[T]) TakeControl(w http.ResponseWriter, r *http.Request, _ views.View) {
	s.Handle(w, r, s.Instance)
}
