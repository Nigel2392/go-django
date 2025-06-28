package internal

type Set[E comparable] map[E]struct{}

func (s Set[E]) Contains(e E) bool {
	_, ok := s[e]
	return ok
}

func (s Set[E]) Add(e ...E) {
	for _, elem := range e {
		s[elem] = struct{}{}
	}
}

func (s Set[E]) Remove(e E) {
	delete(s, e)
}

func (s Set[E]) Clear() {
	for e := range s {
		delete(s, e)
	}
}

func NewSet[E comparable](s ...E) Set[E] {
	if len(s) == 0 {
		return make(Set[E])
	}

	var m = make(Set[E], len(s))
	for _, e := range s {
		m[e] = struct{}{}
	}
	return m
}
