package components

import "github.com/Nigel2392/go-django/src/internal/django_reflect"

type namespace struct {
	name string
	*ComponentRegistry
}

func (n *namespace) Render(name string, args ...interface{}) Component {
	var (
		cmpFn *django_reflect.Func
		cmap  map[string]*django_reflect.Func
		ok    bool
	)
	if cmap, ok = n.ns_components[n.name]; !ok {
		return nil
	}
	if cmpFn, ok = cmap[name]; !ok {
		return nil
	}
	var ret = cmpFn.Call(args...)
	return ret[0].(Component)
}

func (n *namespace) Register(name string, componentFn ComponentFunc) {
	if _, ok := n.ns_components[n.name]; !ok {
		n.ns_components[n.name] = make(map[string]*django_reflect.Func)
	}

	n.ns_components[n.name][name] = n.newComponent(componentFn)
}
