package internal

var _ Set[any] = (*OrderedSet[any])(nil)

type setKey[T any] struct {
	val  T
	next *setKey[T]
	prev *setKey[T]
}

type OrderedSet[E comparable] struct {
	head *setKey[E]
	tail *setKey[E]
	size int
	set  map[E]*setKey[E]
}

func NewOrderedSet[E comparable](elements ...E) *OrderedSet[E] {
	var s = &OrderedSet[E]{}
	s.set = make(map[E]*setKey[E], len(elements))
	for _, e := range elements {
		s.Add(e)
	}
	return s
}

func (s *OrderedSet[E]) Contains(e E) bool {
	if s.set == nil {
		return false
	}
	_, exists := s.set[e]
	return exists
}

func (s *OrderedSet[E]) Add(e ...E) {
	if s.set == nil {
		s.set = make(map[E]*setKey[E])
	}

	for _, elem := range e {
		s.add(elem)
	}
}

func (s *OrderedSet[E]) add(e E) {
	if _, exists := s.set[e]; exists {
		return
	}

	var newKey = &setKey[E]{val: e}
	if s.head == nil {
		s.head = newKey
		s.tail = newKey
	} else {
		s.tail.next = newKey
		newKey.prev = s.tail
		s.tail = newKey
	}

	s.set[e] = newKey
	s.size++
}

func (s *OrderedSet[E]) Remove(e E) {
	if s.set == nil {
		return
	}
	if key, exists := s.set[e]; exists {
		if key.prev != nil {
			key.prev.next = key.next
		} else {
			s.head = key.next
		}
		if key.next != nil {
			key.next.prev = key.prev
		} else {
			s.tail = key.prev
		}

		// be nice to the garbage collector
		// unsure if this provides any benefit
		key.prev = nil
		key.next = nil

		// delete from map
		delete(s.set, e)
		s.size--
	}
}

func (s *OrderedSet[E]) Clear() {
	if s.set == nil {
		return
	}
	s.head = nil
	s.tail = nil
	s.set = make(map[E]*setKey[E])
	s.size = 0
}

func (s *OrderedSet[E]) Len() int {
	return s.size
}

func (s *OrderedSet[E]) Values() []E {
	if s.set == nil {
		return nil
	}
	var values = make([]E, 0, s.size)
	for key := s.head; key != nil; key = key.next {
		values = append(values, key.val)
	}
	return values
}
