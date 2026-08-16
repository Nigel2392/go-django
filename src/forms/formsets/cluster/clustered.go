package cluster

import (
	"github.com/Nigel2392/go-django/internal/forms"
)

var _ Clusterable = (&ClusteredForm[forms.Minimum]{})

type ClusteredForm[THIS forms.Minimum] struct {
	_THIS
	parent Clusterable
	Order  SAVE_ORDERING
}

func NewClusterable[THIS forms.Minimum](form THIS, saveOrder SAVE_ORDERING) Clusterable {
	return &ClusteredForm[THIS]{
		_THIS: form,
		Order: saveOrder,
	}
}

func (b *ClusteredForm[T]) Form() T {
	return b._THIS.(T)
}

func (b *ClusteredForm[T]) Bind(belongsTo Clusterable) forms.Minimum {
	b.parent = belongsTo
	return b
}

func (b *ClusteredForm[T]) SaveOrder() SAVE_ORDERING {
	return b.Order
}

func (b *ClusteredForm[T]) Parent() Clusterable {
	return b.parent
}

func (a *ClusteredForm[T]) Save() error {
	return forms.SaveForm(a.Context(), a._THIS)
}
