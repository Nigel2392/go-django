package attrs

import (
	"fmt"
	"reflect"
)

type node struct {
	meta     *modelMeta
	deps     []*node
	visited  bool
	visiting bool
}

func buildDependencyGraph(reg map[reflect.Type]*modelMeta) ([]*node, error) {
	var nodeMap = make(map[reflect.Type]*node)

	for t, meta := range reg {
		nodeMap[t] = &node{meta: meta}
	}

	// Link dependencies
	for _, n := range nodeMap {
		// dependencyLoop:
		for head := n.meta.forward.Front(); head != nil; head = head.Next() {

			through := head.Value.Through()
			if through != nil {
				depKey := reflect.TypeOf(through.Model())
				depNode, ok := nodeMap[depKey]
				if !ok {
					return nil, fmt.Errorf("dependency %q not found for node %T", depKey, n.meta.model)
				}

				n.deps = append(n.deps, depNode)
			}

			depKey := reflect.TypeOf(head.Value.Model())
			depNode, ok := nodeMap[depKey]
			if !ok {
				return nil, fmt.Errorf("dependency %q not found for node %T", depKey, n.meta.model)
			}
			n.deps = append(n.deps, depNode)

		}
	}

	// Topological sort
	var ordered = make([]*node, 0, len(nodeMap))
	var visit func(n *node) error
	visit = func(n *node) error {
		if n.visited {
			return nil
		}
		if n.visiting { // cyclic is ok
			return nil
			// return fmt.Errorf("cyclic dependency detected for model: %s", reflect.TypeOf(n.meta.model))
		}

		n.visiting = true

		for _, dep := range n.deps {
			if err := visit(dep); err != nil {
				return err
			}
		}

		n.visited = true
		n.visiting = false
		ordered = append(ordered, n)
		return nil
	}

	for _, n := range nodeMap {
		if !n.visited {
			if err := visit(n); err != nil {
				return nil, err
			}
		}
	}

	return ordered, nil
}
