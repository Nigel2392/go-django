package urls

import core "github.com/Nigel2392/django/core"

type URLFunc struct {
	Func func(core.Mux)
}

func Func(f func(core.Mux)) *URLFunc {
	return &URLFunc{Func: f}
}

func (f *URLFunc) Register(m core.Mux) {
	f.Func(m)
}
