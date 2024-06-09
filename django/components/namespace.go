package components

type namespace struct {
	name string
	*ComponentRegistry
}

func (n *namespace) Render(name string, args ...interface{}) Component {
	var (
		cmpFn *reflectFunc
		cmap  map[string]*reflectFunc
		ok    bool
	)
	if cmap, ok = n.ns_components[n.name]; !ok {
		return nil
	}
	if cmpFn, ok = cmap[name]; !ok {
		return nil
	}
	return cmpFn.Call(args...).(Component)
}

func (n *namespace) Register(name string, componentFn ComponentFunc) {
	if _, ok := n.ns_components[n.name]; !ok {
		n.ns_components[n.name] = make(map[string]*reflectFunc)
	}

	n.ns_components[n.name][name] = n.newComponent(componentFn)
}
