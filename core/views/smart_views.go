package views

import (
	"github.com/Nigel2392/go-django/core/models/modelutils/namer"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3"
)

//
//type AdminsiteModel interface {
//	ID() int64
//	ListDisplay() []string
//	DetailView() Form
//	CreateView() Form
//}

type CrudModel[T any] interface {
	// Reader
	interfaces.Deleter
	interfaces.Saver
	interfaces.Lister[T]
}

type CRUDView[T CrudModel[T]] struct {
	// DeleteView
	Delete *DeleteView[T]

	// UpdateView
	Update *UpdateView[T]

	// CreateView
	Create *CreateView[T]

	// ListView
	List *ListView[T]

	// DetailView
	Detail *DetailView[T]
}

func (c *CRUDView[T]) URLs() router.Registrar {
	var name = namer.GetModelName(*new(T))
	var urls = router.Group("", name)
	if c.List != nil {
		urls.Get("/", c.List.ServeHTTP, "list")
	}
	if c.Create != nil {
		// urls.Get("/create", c.Create.ServeHTTP, "create")
	}
	if c.Detail != nil {
		urls.Get("/<<id>>", c.Detail.ServeHTTP, "detail")
	}
	if c.Update != nil {
		// urls.Get("/<<id>>/update", c.Update.ServeHTTP, "update")
	}
	if c.Delete != nil {
		urls.Get("/<<id>>/delete", c.Delete.ServeHTTP, "delete")
	}
	return urls
}

//
//	type Serializer interface {
//		Serialize(interface{}) ([]byte, error)
//		Deserialize([]byte, interface{}) error
//	}
//
//	type APICrudView[T CrudModel] struct {
//		Serializer Serializer
//	}
